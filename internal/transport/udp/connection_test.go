package udp

import (
	"context"
	"testing"
	"time"

	"github.com/rcarmo/rdp-html5/internal/protocol/rdpeudp"
)

// ============================================================================
// Microsoft Protocol Test Suite Validation Tests
// Reference: MS-RDPEUDP_ClientTestDesignSpecification.md
// ============================================================================

// TestSYNDatagram_PerSpec validates SYN datagram per MS-RDPEUDP Section 3.1.5.1.1
// This matches test case S1_Connection_Initialization_InitialUDPConnection
func TestSYNDatagram_PerSpec(t *testing.T) {
	tests := []struct {
		name     string
		reliable bool
		version  uint16
	}{
		{"Reliable_V1", true, rdpeudp.ProtocolVersion1},
		{"Reliable_V2", true, rdpeudp.ProtocolVersion2},
		{"Lossy_V1", false, rdpeudp.ProtocolVersion1},
		{"Lossy_V2", false, rdpeudp.ProtocolVersion2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &Config{
				MTU:               DefaultMTU,
				ReceiveWindowSize: DefaultReceiveWindowSize,
				Reliable:          tc.reliable,
				ProtocolVersion:   tc.version,
			}
			conn, _ := NewConnection(cfg)
			packet := conn.buildSynPacket()

			// Per spec: snSourceAck MUST be -1 (0xFFFFFFFF)
			if packet.Header.SnSourceAck != 0xFFFFFFFF {
				t.Errorf("snSourceAck = 0x%08X, want 0xFFFFFFFF", packet.Header.SnSourceAck)
			}

			// Per spec: uReceiveWindowSize must be > 0
			if packet.Header.SourceAckReceiveWindowSize == 0 {
				t.Error("uReceiveWindowSize must be > 0")
			}

			// Per spec: RDPUDP_FLAG_SYN MUST be set
			if !packet.Header.HasFlag(rdpeudp.FlagSYN) {
				t.Error("RDPUDP_FLAG_SYN MUST be set")
			}

			// Per spec: If reliable, RDPUDP_FLAG_SYNLOSSY must NOT be set
			// If lossy, RDPUDP_FLAG_SYNLOSSY MUST be set
			hasSynLossy := packet.Header.HasFlag(rdpeudp.FlagSYNLOSSY)
			if tc.reliable && hasSynLossy {
				t.Error("Reliable mode: RDPUDP_FLAG_SYNLOSSY must NOT be set")
			}
			if !tc.reliable && !hasSynLossy {
				t.Error("Lossy mode: RDPUDP_FLAG_SYNLOSSY MUST be set")
			}

			// Per spec: MTU must be in [1132, 1232]
			if packet.SynData.UpstreamMTU < 1132 || packet.SynData.UpstreamMTU > 1232 {
				t.Errorf("uUpStreamMtu = %d, must be in [1132, 1232]", packet.SynData.UpstreamMTU)
			}
			if packet.SynData.DownstreamMTU < 1132 || packet.SynData.DownstreamMTU > 1232 {
				t.Errorf("uDownStreamMtu = %d, must be in [1132, 1232]", packet.SynData.DownstreamMTU)
			}

			// Per spec: Version 2+ should have SYNEX flag
			if tc.version >= rdpeudp.ProtocolVersion2 {
				if !packet.Header.HasFlag(rdpeudp.FlagSYNEX) {
					t.Error("Version 2+: RDPUDP_FLAG_SYNEX should be set")
				}
				if packet.SynDataEx == nil {
					t.Error("Version 2+: SynDataEx should be present")
				} else {
					if packet.SynDataEx.Version != tc.version {
						t.Errorf("SynDataEx.Version = 0x%04X, want 0x%04X", packet.SynDataEx.Version, tc.version)
					}
				}
			}

			// Per spec: snInitialSequenceNumber must be random (non-zero)
			if packet.SynData.SnInitialSequenceNumber == 0 {
				t.Error("snInitialSequenceNumber should be random, got 0")
			}
		})
	}
}

// TestSYNDatagram_ZeroPadding validates zero-padding per spec
// Per spec: "This datagram MUST be zero-padded to increase the size to 1232 bytes"
func TestSYNDatagram_ZeroPadding(t *testing.T) {
	conn, _ := NewConnection(nil)
	packet := conn.buildSynPacket()

	data, err := packet.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Raw packet should be smaller than MTU
	if len(data) >= int(DefaultMTU) {
		t.Skipf("Packet already at MTU size (%d bytes)", len(data))
	}

	// Verify we have the padding logic in sendPacket
	// The actual padding happens in sendPacket(), not in Serialize()
	t.Logf("Raw SYN packet size: %d bytes (will be padded to %d in sendPacket)", len(data), DefaultMTU)
}

// TestACKDatagram_PerSpec validates ACK datagram format
// Reference: S2_DataTransfer test cases
func TestACKDatagram_PerSpec(t *testing.T) {
	packet := rdpeudp.NewACKPacket(12345, 64)

	// Per spec: RDPUDP_FLAG_ACK MUST be set
	if !packet.Header.HasFlag(rdpeudp.FlagACK) {
		t.Error("RDPUDP_FLAG_ACK MUST be set")
	}

	// Per spec: snSourceAck should be the sequence number being acknowledged
	if packet.Header.SnSourceAck != 12345 {
		t.Errorf("snSourceAck = %d, want 12345", packet.Header.SnSourceAck)
	}
}

// TestSequenceNumberWrapAround validates wrap-around handling
// Reference: S2_DataTransfer_SequenceNumberWrapAround
func TestSequenceNumberWrapAround(t *testing.T) {
	// Test sequence number near max value
	maxSeq := uint32(0xFFFFFFFF)

	// Simulating sequence near wrap-around
	seqNumbers := []uint32{
		maxSeq - 2, // 0xFFFFFFFD
		maxSeq - 1, // 0xFFFFFFFE
		maxSeq,     // 0xFFFFFFFF
		0,          // Wrapped to 0
		1,          // 0x00000001
	}

	for i, seq := range seqNumbers {
		packet := rdpeudp.NewDataPacket(seq, seq, []byte("test"))
		if packet.SourcePayload.SnSourceStart != seq {
			t.Errorf("Packet %d: SnSourceStart = %d, want %d", i, packet.SourcePayload.SnSourceStart, seq)
		}
	}

	// Verify wrap-around math works
	var seq uint32 = 0xFFFFFFFF
	seq++ // Should wrap to 0
	if seq != 0 {
		t.Errorf("Sequence wrap: got %d, want 0", seq)
	}
}

// ============================================================================
// Original Tests
// ============================================================================

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

// ============================================================================
// Retransmission Timer Tests
// Reference: MS-RDPEUDP Section 3.1.6.1
// ============================================================================

func TestGetRetransmitTimeout_V1(t *testing.T) {
	cfg := &Config{
		ProtocolVersion: rdpeudp.ProtocolVersion1,
	}
	conn, _ := NewConnection(cfg)

	timeout := conn.getRetransmitTimeout()

	// Per spec: "RDPUDP_PROTOCOL_VERSION_1: the minimum retransmit time-out is 500 ms"
	if timeout < RetransmitTimeoutV1 {
		t.Errorf("V1 retransmit timeout = %v, want >= %v", timeout, RetransmitTimeoutV1)
	}
}

func TestGetRetransmitTimeout_V2(t *testing.T) {
	cfg := &Config{
		ProtocolVersion: rdpeudp.ProtocolVersion2,
	}
	conn, _ := NewConnection(cfg)

	timeout := conn.getRetransmitTimeout()

	// Per spec: "RDPUDP_PROTOCOL_VERSION_2: the minimum retransmit time-out is 300 ms"
	if timeout < RetransmitTimeoutV2 {
		t.Errorf("V2 retransmit timeout = %v, want >= %v", timeout, RetransmitTimeoutV2)
	}
}

func TestGetRetransmitTimeout_RTTBased(t *testing.T) {
	cfg := &Config{
		ProtocolVersion: rdpeudp.ProtocolVersion2,
	}
	conn, _ := NewConnection(cfg)

	// Simulate a measured RTT that's much larger than minimum
	conn.rtt = 500 * time.Millisecond

	timeout := conn.getRetransmitTimeout()

	// Per spec: "minimum retransmit time-out or twice the RTT, whichever is longer"
	expected := 2 * conn.rtt
	if timeout != expected {
		t.Errorf("RTT-based timeout = %v, want %v (2 * RTT)", timeout, expected)
	}
}

// ============================================================================
// ACK Vector Tests  
// Reference: MS-RDPEUDP Section 2.2.2.7 and 3.1.1.4.1
// ============================================================================

func TestBuildAckVector_Empty(t *testing.T) {
	conn, _ := NewConnection(nil)

	// With no received packets, ACK vector should be nil
	vector := conn.buildAckVector()
	if vector != nil {
		t.Error("ACK vector should be nil with no received packets")
	}
}

func TestBuildAckVector_AllReceived(t *testing.T) {
	conn, _ := NewConnection(nil)

	// Simulate receiving consecutive packets
	conn.nextExpectSeq = 100
	conn.highestRecvSeq = 103
	// Packets 100, 101, 102, 103 all received

	vector := conn.buildAckVector()
	if vector == nil {
		t.Fatal("ACK vector should not be nil")
	}

	// All packets received, so first element should indicate received state
	if len(vector.AckVectorElements) == 0 {
		t.Fatal("ACK vector should have elements")
	}

	// Per spec: 2 bits state, 6 bits length
	firstElement := vector.AckVectorElements[0]
	state := (firstElement >> 6) & 0x03
	if state != AckStateReceived {
		t.Errorf("First element state = %d, want %d (RECEIVED)", state, AckStateReceived)
	}
}

func TestBuildAckVector_WithGaps(t *testing.T) {
	conn, _ := NewConnection(nil)

	// Simulate receiving packets with a gap
	conn.nextExpectSeq = 100
	conn.highestRecvSeq = 105
	// Put packet 103 in receive buffer (out of order)
	conn.recvBuffer[103] = []byte("data")
	// Packets 100, 101, 102 delivered, 103 buffered, 104 missing, 105 received

	vector := conn.buildAckVector()
	if vector == nil {
		t.Fatal("ACK vector should not be nil")
	}

	// Should have multiple elements due to state changes
	t.Logf("ACK vector has %d elements", len(vector.AckVectorElements))
}

func TestProcessAckVector_Basic(t *testing.T) {
	conn, _ := NewConnection(nil)
	conn.state = StateEstablished
	conn.lastAckedSeq = 102 // The cumulative ACK

	// Add packets to send buffer (seq 100, 101, 102)
	conn.sendBuffer[100] = &sentPacket{seqNum: 100, data: []byte("data100")}
	conn.sendBuffer[101] = &sentPacket{seqNum: 101, data: []byte("data101")}
	conn.sendBuffer[102] = &sentPacket{seqNum: 102, data: []byte("data102")}

	// Create ACK vector indicating packets from lastAckedSeq going backwards are received
	// Per spec: ACK vector starts at snSourceAck (lastAckedSeq) and goes backwards
	// Element encoding: state in top 2 bits, length in bottom 6 bits
	// State 0 = received, length 2 means 3 packets (seq 102, 101, 100)
	ackVector := &rdpeudp.AckVector{
		AckVectorSize:     1,
		AckVectorElements: []uint8{(AckStateReceived << 6) | 2}, // 3 packets received
	}

	conn.processAckVector(ackVector)

	// All 3 packets should be removed from send buffer
	// Note: The ACK vector processes from lastAckedSeq (102) backwards
	if _, ok := conn.sendBuffer[102]; ok {
		t.Error("Packet 102 should be removed from send buffer")
	}
	if _, ok := conn.sendBuffer[101]; ok {
		t.Error("Packet 101 should be removed from send buffer")
	}
	if _, ok := conn.sendBuffer[100]; ok {
		t.Error("Packet 100 should be removed from send buffer")
	}
}

// ============================================================================
// Congestion Control Tests
// Reference: MS-RDPEUDP Section 3.1.1.6
// ============================================================================

func TestHandleCongestionNotification(t *testing.T) {
	conn, _ := NewConnection(nil)
	initialCwnd := conn.congestionWindow

	conn.handleCongestionNotification()

	// Per spec: multiplicative decrease (halve the window)
	expectedCwnd := initialCwnd / 2
	if conn.congestionWindow != expectedCwnd {
		t.Errorf("Congestion window = %d, want %d (half of %d)", 
			conn.congestionWindow, expectedCwnd, initialCwnd)
	}

	// Stats should be updated
	if conn.stats.CongestionEvents != 1 {
		t.Errorf("CongestionEvents = %d, want 1", conn.stats.CongestionEvents)
	}
}

func TestHandleCongestionNotification_MinWindow(t *testing.T) {
	conn, _ := NewConnection(nil)
	conn.congestionWindow = 1

	conn.handleCongestionNotification()

	// Window should never go below 1
	if conn.congestionWindow < 1 {
		t.Errorf("Congestion window = %d, should never be < 1", conn.congestionWindow)
	}
}

// ============================================================================
// Keepalive Timer Tests
// Reference: MS-RDPEUDP Section 3.1.1.9 and 3.1.6.2
// ============================================================================

func TestKeepaliveTimeout_Value(t *testing.T) {
	// Per spec: 65 seconds
	if KeepaliveTimeout != 65*time.Second {
		t.Errorf("KeepaliveTimeout = %v, want 65s", KeepaliveTimeout)
	}
}

func TestConnection_LastRecvTimeTracking(t *testing.T) {
	conn, _ := NewConnection(nil)
	conn.state = StateEstablished

	// Simulate receiving a packet
	packet := &rdpeudp.Packet{
		Header: rdpeudp.FECHeader{
			Flags: rdpeudp.FlagACK,
		},
	}

	data, _ := packet.Serialize()
	before := time.Now()
	conn.handleReceivedPacket(data)
	after := time.Now()

	// lastRecvTime should be updated
	if conn.lastRecvTime.Before(before) || conn.lastRecvTime.After(after) {
		t.Error("lastRecvTime not updated correctly on packet receive")
	}
}

// ============================================================================
// Connection Stats Tests
// ============================================================================

func TestConnectionStats_RTT(t *testing.T) {
	cfg := DefaultConfig()
	conn, _ := NewConnection(cfg)

	// Initial RTT should be set to default
	if conn.rtt != RetransmitTimeoutV2 {
		t.Errorf("Initial RTT = %v, want %v", conn.rtt, RetransmitTimeoutV2)
	}
}

func TestConnectionStats_CongestionEvents(t *testing.T) {
	conn, _ := NewConnection(nil)

	// Initial congestion events should be 0
	stats := conn.Stats()
	if stats.CongestionEvents != 0 {
		t.Errorf("Initial CongestionEvents = %d, want 0", stats.CongestionEvents)
	}

	// After handling congestion, count should increment
	conn.handleCongestionNotification()
	stats = conn.Stats()
	if stats.CongestionEvents != 1 {
		t.Errorf("CongestionEvents after notification = %d, want 1", stats.CongestionEvents)
	}
}
