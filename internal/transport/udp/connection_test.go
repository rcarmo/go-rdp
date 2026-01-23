package udp

import (
	"context"
	"testing"
	"time"
)

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "CLOSED"},
		{StateListen, "LISTEN"},
		{StateSynSent, "SYN_SENT"},
		{StateSynReceived, "SYN_RECEIVED"},
		{StateEstablished, "ESTABLISHED"},
		{State(99), "UNKNOWN(99)"},
	}

	for _, tc := range tests {
		if tc.state.String() != tc.expected {
			t.Errorf("State(%d).String() = %q, want %q", tc.state, tc.state.String(), tc.expected)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MTU != DefaultMTU {
		t.Errorf("MTU = %d, want %d", cfg.MTU, DefaultMTU)
	}
	if cfg.ReceiveWindowSize != DefaultReceiveWindowSize {
		t.Errorf("ReceiveWindowSize = %d, want %d", cfg.ReceiveWindowSize, DefaultReceiveWindowSize)
	}
	if !cfg.Reliable {
		t.Error("Reliable should be true by default")
	}
}

func TestNewConnection(t *testing.T) {
	conn, err := NewConnection(nil)
	if err != nil {
		t.Fatalf("NewConnection failed: %v", err)
	}

	if conn.State() != StateClosed {
		t.Errorf("Initial state = %v, want CLOSED", conn.State())
	}

	// Should have a random initial sequence number
	if conn.localSeqNum == 0 {
		t.Error("Initial sequence number should not be 0")
	}
}

func TestNewConnection_WithConfig(t *testing.T) {
	cfg := &Config{
		MTU:               MinMTU,
		ReceiveWindowSize: 32,
		Reliable:          false,
	}

	conn, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("NewConnection failed: %v", err)
	}

	if conn.config.MTU != MinMTU {
		t.Errorf("MTU = %d, want %d", conn.config.MTU, MinMTU)
	}
	if conn.config.ReceiveWindowSize != 32 {
		t.Errorf("ReceiveWindowSize = %d, want 32", conn.config.ReceiveWindowSize)
	}
}

func TestNewConnection_MTUValidation(t *testing.T) {
	// MTU too low should be corrected
	cfg := &Config{MTU: 100}
	conn, err := NewConnection(cfg)
	if err != nil {
		t.Fatalf("NewConnection failed: %v", err)
	}
	if conn.config.MTU != DefaultMTU {
		t.Errorf("Invalid MTU should be corrected to default, got %d", conn.config.MTU)
	}

	// MTU too high should be corrected
	cfg = &Config{MTU: 2000}
	conn, err = NewConnection(cfg)
	if err != nil {
		t.Fatalf("NewConnection failed: %v", err)
	}
	if conn.config.MTU != DefaultMTU {
		t.Errorf("Invalid MTU should be corrected to default, got %d", conn.config.MTU)
	}
}

func TestConnection_Stats(t *testing.T) {
	conn, _ := NewConnection(nil)
	stats := conn.Stats()

	if stats.PacketsSent != 0 {
		t.Error("Initial stats should be zero")
	}
	if stats.PacketsReceived != 0 {
		t.Error("Initial stats should be zero")
	}
}

func TestConnection_ConnectWithoutRemoteAddr(t *testing.T) {
	conn, _ := NewConnection(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := conn.Connect(ctx)
	if err == nil {
		t.Error("Connect without RemoteAddr should fail")
	}
}

func TestConnection_ConnectInvalidState(t *testing.T) {
	conn, _ := NewConnection(nil)
	conn.state = StateEstablished // Simulate already connected

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := conn.Connect(ctx)
	if err != ErrInvalidState {
		t.Errorf("Connect in wrong state should return ErrInvalidState, got %v", err)
	}
}

func TestConnection_WriteInvalidState(t *testing.T) {
	conn, _ := NewConnection(nil)

	_, err := conn.Write([]byte("test"))
	if err != ErrInvalidState {
		t.Errorf("Write when not established should return ErrInvalidState, got %v", err)
	}
}

func TestConnection_CloseIdempotent(t *testing.T) {
	conn, _ := NewConnection(nil)

	// First close should succeed
	if err := conn.Close(); err != nil {
		t.Errorf("First Close failed: %v", err)
	}

	// Second close should also succeed (idempotent)
	if err := conn.Close(); err != nil {
		t.Errorf("Second Close failed: %v", err)
	}
}

func TestGenerateInitialSequenceNumber(t *testing.T) {
	// Generate multiple sequence numbers and ensure they're different
	seen := make(map[uint32]bool)
	for i := 0; i < 100; i++ {
		seq := generateInitialSequenceNumber()
		if seen[seq] {
			t.Errorf("Duplicate sequence number generated: %d", seq)
		}
		seen[seq] = true
	}
}

func TestMinUint16(t *testing.T) {
	tests := []struct {
		a, b, want uint16
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 100, 0},
	}

	for _, tc := range tests {
		if got := minUint16(tc.a, tc.b); got != tc.want {
			t.Errorf("minUint16(%d, %d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestConnection_LocalRemoteAddr(t *testing.T) {
	conn, _ := NewConnection(nil)

	// Without a socket, addresses should be nil
	if conn.LocalAddr() != nil {
		t.Error("LocalAddr should be nil before connection")
	}
	if conn.RemoteAddr() != nil {
		t.Error("RemoteAddr should be nil before connection")
	}
}

func TestBuildSynPacket_Reliable(t *testing.T) {
	cfg := &Config{
		MTU:               DefaultMTU,
		ReceiveWindowSize: 64,
		Reliable:          true,
		ProtocolVersion:   0x0002, // Version 2
	}
	conn, _ := NewConnection(cfg)

	packet := conn.buildSynPacket()

	// Should have SYN flag
	if packet.Header.Flags&0x0001 == 0 {
		t.Error("SYN packet should have SYN flag")
	}

	// Should NOT have SYNLOSSY flag (reliable mode)
	if packet.Header.Flags&0x0200 != 0 {
		t.Error("Reliable SYN should not have SYNLOSSY flag")
	}

	// Should have SYNEX flag for version 2
	if packet.Header.Flags&0x1000 == 0 {
		t.Error("Version 2+ SYN should have SYNEX flag")
	}
}

func TestBuildSynPacket_Lossy(t *testing.T) {
	cfg := &Config{
		MTU:               DefaultMTU,
		ReceiveWindowSize: 64,
		Reliable:          false, // Lossy mode
		ProtocolVersion:   0x0002,
	}
	conn, _ := NewConnection(cfg)

	packet := conn.buildSynPacket()

	// Should have SYNLOSSY flag
	if packet.Header.Flags&0x0200 == 0 {
		t.Error("Lossy SYN should have SYNLOSSY flag")
	}
}
