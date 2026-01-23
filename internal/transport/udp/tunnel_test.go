package udp

import (
	"testing"
	"time"

	"github.com/rcarmo/rdp-html5/internal/protocol/rdpemt"
)

// TestTunnelState_String validates tunnel state string representations
func TestTunnelState_String(t *testing.T) {
	tests := []struct {
		state    TunnelState
		expected string
	}{
		{TunnelStateInit, "INIT"},
		{TunnelStateConnecting, "CONNECTING"},
		{TunnelStateSecuring, "SECURING"},
		{TunnelStateTunnelCreate, "TUNNEL_CREATE"},
		{TunnelStateEstablished, "ESTABLISHED"},
		{TunnelStateClosed, "CLOSED"},
		{TunnelStateFailed, "FAILED"},
		{TunnelState(99), "UNKNOWN(99)"},
	}

	for _, tc := range tests {
		if tc.state.String() != tc.expected {
			t.Errorf("TunnelState(%d).String() = %q, want %q", tc.state, tc.state.String(), tc.expected)
		}
	}
}

// TestNewTunnelManager validates tunnel manager creation
func TestNewTunnelManager(t *testing.T) {
	tests := []struct {
		name    string
		config  *TunnelManagerConfig
		wantErr bool
	}{
		{
			name:    "NilConfig",
			config:  nil,
			wantErr: false,
		},
		{
			name: "ValidConfig",
			config: &TunnelManagerConfig{
				Enabled:         true,
				ConnectTimeout:  5 * time.Second,
				ProtocolVersion: 0x0002,
			},
			wantErr: false,
		},
		{
			name: "WithServerAddr",
			config: &TunnelManagerConfig{
				ServerAddr:     "192.168.1.1:3389",
				Enabled:        true,
				ConnectTimeout: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "InvalidServerAddr",
			config: &TunnelManagerConfig{
				ServerAddr: "invalid:addr:format::::",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tm, err := NewTunnelManager(tc.config)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if tm == nil {
					t.Error("expected non-nil TunnelManager")
				}
			}
		})
	}
}

// TestTunnelManager_SetServerAddr validates server address setting
func TestTunnelManager_SetServerAddr(t *testing.T) {
	tm, _ := NewTunnelManager(nil)

	// Valid address
	err := tm.SetServerAddr("192.168.1.1", 3389)
	if err != nil {
		t.Errorf("SetServerAddr error: %v", err)
	}

	// Invalid host should fail
	err = tm.SetServerAddr("", -1)
	if err == nil {
		t.Error("expected error for invalid address")
	}
}

// TestTunnelManager_SetEnabled validates enable/disable
func TestTunnelManager_SetEnabled(t *testing.T) {
	tm, _ := NewTunnelManager(nil)

	// Initially disabled
	if tm.IsEnabled() {
		t.Error("should be disabled by default")
	}

	// Enable
	tm.SetEnabled(true)
	if !tm.IsEnabled() {
		t.Error("should be enabled after SetEnabled(true)")
	}

	// Disable
	tm.SetEnabled(false)
	if tm.IsEnabled() {
		t.Error("should be disabled after SetEnabled(false)")
	}
}

// TestTunnelManager_GetTunnel validates tunnel lookup
func TestTunnelManager_GetTunnel(t *testing.T) {
	tm, _ := NewTunnelManager(nil)

	// Non-existent tunnel
	tunnel := tm.GetTunnel(12345)
	if tunnel != nil {
		t.Error("expected nil for non-existent tunnel")
	}
}

// TestTunnelManager_HandleMultitransportRequest_Disabled validates disabled handling
func TestTunnelManager_HandleMultitransportRequest_Disabled(t *testing.T) {
	tm, _ := NewTunnelManager(nil)
	tm.SetEnabled(false)

	req := &rdpemt.MultitransportRequest{
		RequestID:         12345,
		RequestedProtocol: rdpemt.ProtocolUDPFECReliable,
	}

	// Should return nil (will be declined by caller)
	err := tm.HandleMultitransportRequest(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Tunnel should NOT be created when disabled
	tunnel := tm.GetTunnel(12345)
	if tunnel != nil {
		t.Error("tunnel should not be created when disabled")
	}
}

// TestTunnelManager_HandleMultitransportRequest_NoServerAddr validates error without server address
func TestTunnelManager_HandleMultitransportRequest_NoServerAddr(t *testing.T) {
	tm, _ := NewTunnelManager(nil)
	tm.SetEnabled(true)
	// No server address set

	req := &rdpemt.MultitransportRequest{
		RequestID:         12345,
		RequestedProtocol: rdpemt.ProtocolUDPFECReliable,
	}

	err := tm.HandleMultitransportRequest(req)
	if err == nil {
		t.Error("expected error when server address not set")
	}
}

// TestTunnelManager_CloseAll validates closing all tunnels
func TestTunnelManager_CloseAll(t *testing.T) {
	tm, _ := NewTunnelManager(nil)

	// CloseAll should not panic when empty
	tm.CloseAll()
}

// TestTunnelManager_SendData_NoTunnel validates sending to non-existent tunnel
func TestTunnelManager_SendData_NoTunnel(t *testing.T) {
	tm, _ := NewTunnelManager(nil)

	err := tm.SendData(12345, []byte("test"))
	if err == nil {
		t.Error("expected error for non-existent tunnel")
	}
}

// TestTunnelManager_CloseTunnel_NoTunnel validates closing non-existent tunnel
func TestTunnelManager_CloseTunnel_NoTunnel(t *testing.T) {
	tm, _ := NewTunnelManager(nil)

	err := tm.CloseTunnel(12345)
	if err == nil {
		t.Error("expected error for non-existent tunnel")
	}
}

// TestTunnel_State validates tunnel state access
func TestTunnel_State(t *testing.T) {
	tunnel := &Tunnel{
		state: TunnelStateInit,
	}

	if tunnel.State() != TunnelStateInit {
		t.Errorf("State() = %v, want INIT", tunnel.State())
	}
}

// TestTunnel_LastError validates error access
func TestTunnel_LastError(t *testing.T) {
	tunnel := &Tunnel{}

	if tunnel.LastError() != nil {
		t.Error("LastError() should be nil initially")
	}
}

// TestTunnel_Write_NotEstablished validates write fails when not established
func TestTunnel_Write_NotEstablished(t *testing.T) {
	tunnel := &Tunnel{
		state: TunnelStateInit,
	}

	err := tunnel.Write([]byte("test"))
	if err != ErrClosed {
		t.Errorf("Write error = %v, want ErrClosed", err)
	}
}

// TestTunnel_Read_Closed validates read fails when closed
func TestTunnel_Read_Closed(t *testing.T) {
	tunnel := &Tunnel{
		dataChan: make(chan []byte, 1),
		closeCh:  make(chan struct{}),
	}

	// Close the tunnel
	close(tunnel.closeCh)

	_, err := tunnel.Read()
	if err != ErrClosed {
		t.Errorf("Read error = %v, want ErrClosed", err)
	}
}

// ============================================================================
// MS-RDPEMT Protocol Test Suite Validation Tests
// Reference: MS-RDPEMT_ServerTestDesignSpecification.md
// ============================================================================

// TestTunnelManager_MultitransportRequest_PositiveTest
// Reference: S1_MultitransportConnectionInitiation_PositiveTest
func TestTunnelManager_MultitransportRequest_PositiveTest(t *testing.T) {
	// This test validates that we correctly handle a multitransport request
	// The actual UDP connection establishment would require a real server

	tm, _ := NewTunnelManager(&TunnelManagerConfig{
		ServerAddr:      "127.0.0.1:3389",
		Enabled:         true,
		ConnectTimeout:  100 * time.Millisecond,
		ProtocolVersion: 0x0002,
	})

	// Test with reliable transport request
	req := &rdpemt.MultitransportRequest{
		RequestID:         0x12345678,
		RequestedProtocol: rdpemt.ProtocolUDPFECReliable,
		SecurityCookie:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}

	err := tm.HandleMultitransportRequest(req)
	if err != nil {
		t.Errorf("HandleMultitransportRequest error: %v", err)
	}

	// Wait a bit for async processing
	time.Sleep(10 * time.Millisecond)

	// Tunnel should be created (even if connection fails)
	tunnel := tm.GetTunnel(0x12345678)
	if tunnel == nil {
		// Tunnel may have been removed due to connection failure
		t.Log("Tunnel not found (expected if no server)")
		return
	}

	// Verify tunnel properties
	if tunnel.RequestID != 0x12345678 {
		t.Errorf("RequestID = 0x%08X, want 0x12345678", tunnel.RequestID)
	}

	if !tunnel.Reliable {
		t.Error("Reliable should be true for ProtocolUDPFECReliable")
	}
}

// TestTunnelManager_MultitransportRequest_LossyTransport
// Tests handling of lossy (DTLS) transport requests
func TestTunnelManager_MultitransportRequest_LossyTransport(t *testing.T) {
	tm, _ := NewTunnelManager(&TunnelManagerConfig{
		ServerAddr:      "127.0.0.1:3389",
		Enabled:         true,
		ConnectTimeout:  100 * time.Millisecond,
		ProtocolVersion: 0x0002,
	})

	req := &rdpemt.MultitransportRequest{
		RequestID:         0xDEADBEEF,
		RequestedProtocol: rdpemt.ProtocolUDPFECLossy, // Lossy = DTLS
		SecurityCookie:    [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	}

	err := tm.HandleMultitransportRequest(req)
	if err != nil {
		t.Errorf("HandleMultitransportRequest error: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	tunnel := tm.GetTunnel(0xDEADBEEF)
	if tunnel == nil {
		t.Log("Tunnel not found (expected if no server)")
		return
	}

	if tunnel.Reliable {
		t.Error("Reliable should be false for ProtocolUDPFECLossy")
	}
}

// TestTunnelManager_SetCallbacks validates callback setting
func TestTunnelManager_SetCallbacks(t *testing.T) {
	tm, _ := NewTunnelManager(nil)

	readyCalled := false
	closedCalled := false
	dataCalled := false

	tm.SetCallbacks(
		func(tunnel *Tunnel) { readyCalled = true },
		func(requestID uint32, err error) { closedCalled = true },
		func(requestID uint32, data []byte) { dataCalled = true },
	)

	// Verify callbacks are stored (we can't test they're called without actual tunnels)
	tm.mu.RLock()
	hasReady := tm.onTunnelReady != nil
	hasClosed := tm.onTunnelClosed != nil
	hasData := tm.onChannelData != nil
	tm.mu.RUnlock()

	if !hasReady {
		t.Error("onTunnelReady callback not set")
	}
	if !hasClosed {
		t.Error("onTunnelClosed callback not set")
	}
	if !hasData {
		t.Error("onChannelData callback not set")
	}

	// Callbacks weren't called yet
	if readyCalled || closedCalled || dataCalled {
		t.Error("callbacks should not be called during SetCallbacks")
	}
}
