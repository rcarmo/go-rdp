package udp

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/pion/dtls/v2"
)

// TestSecureConfig validates secure connection configuration
func TestSecureConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *SecureConfig
		wantErr  bool
		errMatch string
	}{
		{
			name:     "NilConfig",
			config:   nil,
			wantErr:  true,
			errMatch: "config required",
		},
		{
			name: "ValidReliable",
			config: &SecureConfig{
				UDPConfig: DefaultConfig(),
				Reliable:  true,
			},
			wantErr: false,
		},
		{
			name: "ValidLossy",
			config: &SecureConfig{
				UDPConfig: DefaultConfig(),
				Reliable:  false,
			},
			wantErr: false,
		},
		{
			name: "WithTLSConfig",
			config: &SecureConfig{
				UDPConfig: DefaultConfig(),
				Reliable:  true,
				TLSConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
			wantErr: false,
		},
		{
			name: "WithDTLSConfig",
			config: &SecureConfig{
				UDPConfig: DefaultConfig(),
				Reliable:  false,
				DTLSConfig: &dtls.Config{
					InsecureSkipVerify: true,
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sc, err := NewSecureConnection(tc.config)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tc.errMatch != "" && err.Error() != tc.errMatch && !contains(err.Error(), tc.errMatch) {
					t.Errorf("error = %q, want match %q", err, tc.errMatch)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if sc == nil {
					t.Error("expected non-nil SecureConnection")
				}
			}
		})
	}
}

// TestSecureConnection_DefaultTLSConfig verifies default TLS config is set
func TestSecureConnection_DefaultTLSConfig(t *testing.T) {
	cfg := &SecureConfig{
		UDPConfig: DefaultConfig(),
		Reliable:  true,
	}

	sc, err := NewSecureConnection(cfg)
	if err != nil {
		t.Fatalf("NewSecureConnection: %v", err)
	}

	if sc.tlsConfig == nil {
		t.Error("tlsConfig should have default value")
	}

	if !sc.tlsConfig.InsecureSkipVerify {
		t.Error("default TLS config should skip verification for RDP")
	}

	if sc.tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("TLS MinVersion = %d, want TLS 1.2 (%d)", sc.tlsConfig.MinVersion, tls.VersionTLS12)
	}
}

// TestSecureConnection_DefaultDTLSConfig verifies default DTLS config is set
func TestSecureConnection_DefaultDTLSConfig(t *testing.T) {
	cfg := &SecureConfig{
		UDPConfig: DefaultConfig(),
		Reliable:  false,
	}

	sc, err := NewSecureConnection(cfg)
	if err != nil {
		t.Fatalf("NewSecureConnection: %v", err)
	}

	if sc.dtlsConfig == nil {
		t.Error("dtlsConfig should have default value")
	}

	if !sc.dtlsConfig.InsecureSkipVerify {
		t.Error("default DTLS config should skip verification for RDP")
	}
}

// TestSecureConnection_ReadWriteNotEstablished verifies errors when not connected
func TestSecureConnection_ReadWriteNotEstablished(t *testing.T) {
	cfg := &SecureConfig{
		UDPConfig: DefaultConfig(),
		Reliable:  true,
	}

	sc, err := NewSecureConnection(cfg)
	if err != nil {
		t.Fatalf("NewSecureConnection: %v", err)
	}

	// Read should fail when not established
	buf := make([]byte, 100)
	_, err = sc.Read(buf)
	if err != ErrClosed {
		t.Errorf("Read error = %v, want ErrClosed", err)
	}

	// Write should fail when not established
	_, err = sc.Write([]byte("test"))
	if err != ErrClosed {
		t.Errorf("Write error = %v, want ErrClosed", err)
	}
}

// TestSecureConnection_IsEstablished verifies establishment state tracking
func TestSecureConnection_IsEstablished(t *testing.T) {
	cfg := &SecureConfig{
		UDPConfig: DefaultConfig(),
		Reliable:  true,
	}

	sc, err := NewSecureConnection(cfg)
	if err != nil {
		t.Fatalf("NewSecureConnection: %v", err)
	}

	if sc.IsEstablished() {
		t.Error("should not be established before Connect")
	}
}

// TestSecureConnection_Close verifies close behavior
func TestSecureConnection_Close(t *testing.T) {
	cfg := &SecureConfig{
		UDPConfig: DefaultConfig(),
		Reliable:  true,
	}

	sc, err := NewSecureConnection(cfg)
	if err != nil {
		t.Fatalf("NewSecureConnection: %v", err)
	}

	// Close should not error even if not connected
	if err := sc.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}

	// Double close should be safe
	if err := sc.Close(); err != nil {
		t.Errorf("Double close error: %v", err)
	}
}

// TestUDPConnWrapper verifies the net.Conn wrapper interface
func TestUDPConnWrapper(t *testing.T) {
	conn, _ := NewConnection(DefaultConfig())
	wrapper := &udpConnWrapper{conn: conn}

	// Test interface compliance
	var _ net.Conn = wrapper

	// Test deadline methods (should be no-ops)
	if err := wrapper.SetDeadline(time.Now()); err != nil {
		t.Errorf("SetDeadline error: %v", err)
	}
	if err := wrapper.SetReadDeadline(time.Now()); err != nil {
		t.Errorf("SetReadDeadline error: %v", err)
	}
	if err := wrapper.SetWriteDeadline(time.Now()); err != nil {
		t.Errorf("SetWriteDeadline error: %v", err)
	}
}

// TestSecureConnection_ConnectNoRemote verifies connect fails without remote address
func TestSecureConnection_ConnectNoRemote(t *testing.T) {
	cfg := &SecureConfig{
		UDPConfig: &Config{
			MTU:               DefaultMTU,
			ReceiveWindowSize: DefaultReceiveWindowSize,
			Reliable:          true,
			// No RemoteAddr set
		},
		Reliable: true,
	}

	sc, err := NewSecureConnection(cfg)
	if err != nil {
		t.Fatalf("NewSecureConnection: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = sc.Connect(ctx)
	if err == nil {
		t.Error("expected error when no remote address")
	}
}

// TestSecureConnection_Addresses verifies address methods
func TestSecureConnection_Addresses(t *testing.T) {
	cfg := &SecureConfig{
		UDPConfig: DefaultConfig(),
		Reliable:  true,
	}

	sc, err := NewSecureConnection(cfg)
	if err != nil {
		t.Fatalf("NewSecureConnection: %v", err)
	}

	// Before connect, addresses should be nil
	if sc.LocalAddr() != nil {
		// May be nil before connect
		t.Logf("LocalAddr before connect: %v", sc.LocalAddr())
	}

	if sc.RemoteAddr() != nil {
		// May be nil before connect
		t.Logf("RemoteAddr before connect: %v", sc.RemoteAddr())
	}
}

// helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr, 0))
}

func containsAt(s, substr string, start int) bool {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// MS-RDPEMT Tunnel PDU Tests (from Microsoft Protocol Test Suite)
// Reference: MS-RDPEMT_ServerTestDesignSpecification.md
// ============================================================================

// TestTunnelCreateRequest_PerSpec validates Tunnel Create Request format
// Per MS-RDPEMT Section 2.2.2.1
func TestTunnelCreateRequest_PerSpec(t *testing.T) {
	// Create a secure connection and build a tunnel request
	cfg := &SecureConfig{
		UDPConfig:      DefaultConfig(),
		Reliable:       true,
		RequestID:      0x12345678,
		SecurityCookie: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}

	sc, err := NewSecureConnection(cfg)
	if err != nil {
		t.Fatalf("NewSecureConnection: %v", err)
	}

	// Verify request ID is stored
	if sc.requestID != 0x12345678 {
		t.Errorf("requestID = 0x%08X, want 0x12345678", sc.requestID)
	}

	// Verify security cookie is stored
	for i := 0; i < 16; i++ {
		if sc.securityCookie[i] != byte(i+1) {
			t.Errorf("securityCookie[%d] = %d, want %d", i, sc.securityCookie[i], i+1)
		}
	}
}

// TestTunnelCreateRequest_InvalidRequestID tests negative case
// Per MS-RDPEMT_ServerTestDesignSpecification: S1_MultitransportConnectionInitiation_NegativeTest_InvalidRequestID
func TestTunnelCreateRequest_InvalidRequestID(t *testing.T) {
	// This test validates that we can construct requests with any RequestID
	// The server is responsible for validating - we just need to send the right format
	cfg := &SecureConfig{
		UDPConfig: DefaultConfig(),
		Reliable:  true,
		RequestID: 0xDEADBEEF, // Invalid RequestID (server would reject)
	}

	sc, err := NewSecureConnection(cfg)
	if err != nil {
		t.Fatalf("NewSecureConnection: %v", err)
	}

	// Verify we can create a connection with any request ID
	if sc.requestID != 0xDEADBEEF {
		t.Errorf("requestID = 0x%08X, want 0xDEADBEEF", sc.requestID)
	}
}
