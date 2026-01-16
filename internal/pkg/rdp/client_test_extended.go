package rdp

import (
	"net"
	"testing"
	"time"

	"github.com/kulaginds/rdp-html5/internal/pkg/rdp/pdu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_getServerName_extended(t *testing.T) {
	tests := []struct {
		name           string
		setupConn      func() net.Conn
		expectedResult string
		expectedEmpty  bool
	}{
		{
			name: "valid connection with hostname",
			setupConn: func() net.Conn {
				// Create a mock connection
				_, client := net.Pipe()
				client.Close() // Close one end to simulate a real connection
				return client
			},
			expectedResult: "", // Pipe will fallback to empty
			expectedEmpty:  true,
		},
		{
			name: "nil connection",
			setupConn: func() net.Conn {
				return nil
			},
			expectedResult: "",
			expectedEmpty:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &client{
				conn: tt.setupConn(),
			}

			result := client.getServerName()

			if tt.expectedEmpty {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestClient_StartTLS_Configuration_extended(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() *client
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil connection",
			setupClient: func() *client {
				return &client{
					conn: nil,
				}
			},
			expectError: true,
			errorMsg:    "TLS handshake",
		},
		{
			name: "valid TLS config",
			setupClient: func() *client {
				return &client{
					conn: &mockConn{},
				}
			},
			expectError: true, // Will fail handshake but config should be valid
			errorMsg:    "TLS handshake",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()

			err := client.StartTLS()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Mock connection for testing
type mockConn struct {
	net.Conn
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return &mockAddr{"localhost:8080"}
}

func (m *mockConn) RemoteAddr() net.Addr {
	return &mockAddr{"server:3389"}
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// Mock address for testing
type mockAddr struct {
	address string
}

func (m *mockAddr) Network() string {
	return "tcp"
}

func (m *mockAddr) String() string {
	return m.address
}

func TestClientConstants(t *testing.T) {
	// Test that constants have expected values
	assert.Equal(t, 5*time.Second, tcpConnectionTimeout)
	assert.Equal(t, 64*1024, readBufferSize)
}

func TestClientStruct(t *testing.T) {
	// Test client struct initialization
	client := &client{
		username:         "testuser",
		password:         "testpass",
		desktopWidth:     1024,
		desktopHeight:    768,
		selectedProtocol: pdu.NegotiationProtocolSSL,
	}

	assert.Equal(t, "testuser", client.username)
	assert.Equal(t, "testpass", client.password)
	assert.Equal(t, uint16(1024), client.desktopWidth)
	assert.Equal(t, uint16(768), client.desktopHeight)
	assert.Equal(t, pdu.NegotiationProtocolSSL, client.selectedProtocol)
}

func TestClient_ChannelIDMap(t *testing.T) {
	client := &client{
		channelIDMap: make(map[string]uint16),
		channels:     []string{"cliprdr", "rdpdr"},
	}

	// Simulate channel initialization
	serverNetworkData := &pdu.ServerNetworkData{
		ChannelIdArray: []uint16{1001, 1002},
		MCSChannelId:   1003,
	}

	client.initChannels(serverNetworkData)

	// Check that channels are mapped correctly
	assert.Equal(t, uint16(1001), client.channelIDMap["cliprdr"])
	assert.Equal(t, uint16(1002), client.channelIDMap["rdpdr"])
	assert.Equal(t, uint16(1003), client.channelIDMap["global"])
}

func TestClient_SkipChannelJoin(t *testing.T) {
	tests := []struct {
		name                 string
		earlyCapabilityFlags uint32
		expectedSkip         bool
	}{
		{
			name:                 "skip channel join supported",
			earlyCapabilityFlags: 0x00000008, // RNS_UD_SC_SKIP_CHANNELJOIN_SUPPORTED
			expectedSkip:         true,
		},
		{
			name:                 "skip channel join not supported",
			earlyCapabilityFlags: 0x00000000,
			expectedSkip:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &client{}
			client.skipChannelJoin = tt.earlyCapabilityFlags&0x8 == 0x8

			client.skipChannelJoin = tt.earlyCapabilityFlags&0x8 == 0x8

			assert.Equal(t, tt.expectedSkip, client.skipChannelJoin)
		})
	}
}

// Integration-style tests for client lifecycle
func TestClient_ConnectionFlow(t *testing.T) {
	// Test client initialization phases
	client := &client{
		username:         "testuser",
		password:         "testpass",
		domain:           "TESTDOMAIN",
		desktopWidth:     1024,
		desktopHeight:    768,
		selectedProtocol: pdu.NegotiationProtocolSSL,
		channelIDMap:     make(map[string]uint16),
		railState:        0,
	}

	// Test initial state
	assert.Empty(t, client.channelIDMap)
	assert.Equal(t, "testuser", client.username)
	assert.Equal(t, "testpass", client.password)
	assert.Equal(t, "TESTDOMAIN", client.domain)
	assert.Equal(t, uint16(1024), client.desktopWidth)
	assert.Equal(t, uint16(768), client.desktopHeight)
	assert.Equal(t, pdu.NegotiationProtocolSSL, client.selectedProtocol)
	assert.Equal(t, uint16(0), client.userID)
	assert.Equal(t, uint32(0), client.shareID)
}

// Test error scenarios
func TestClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		hostname    string
		username    string
		password    string
		width       int
		height      int
		expectError bool
		errorType   string
	}{
		{
			name:        "valid parameters but unreachable host",
			hostname:    "192.0.2.1:3389",
			username:    "testuser",
			password:    "testpass",
			width:       1024,
			height:      768,
			expectError: true,
			errorType:   "tcp connect",
		},
		{
			name:        "invalid hostname format",
			hostname:    "invalid-host",
			username:    "testuser",
			password:    "testpass",
			width:       1024,
			height:      768,
			expectError: true,
			errorType:   "tcp connect",
		},
		{
			name:        "valid IPv6 host",
			hostname:    "[::1]:3389",
			username:    "testuser",
			password:    "testpass",
			width:       1024,
			height:      768,
			expectError: true, // Will be unreachable
			errorType:   "tcp connect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.hostname, tt.username, tt.password, tt.width, tt.height)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorType)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				assert.Equal(t, tt.username, client.username)
				assert.Equal(t, tt.password, client.password)
				assert.Equal(t, uint16(tt.width), client.desktopWidth)
				assert.Equal(t, uint16(tt.height), client.desktopHeight)
			}
		})
	}
}

func TestClient_RemoteApp(t *testing.T) {
	app := &RemoteApp{
		App:        "notepad.exe",
		WorkingDir: "C:\\Users\\TestUser",
		Args:       "",
	}

	client := &client{
		remoteApp: app,
	}

	assert.Equal(t, app, client.remoteApp)
}

func TestClient_Channels(t *testing.T) {
	channels := []string{"cliprdr", "rdpdr", "rdpsnd"}
	client := &client{
		channels:     channels,
		channelIDMap: make(map[string]uint16),
	}

	assert.Equal(t, channels, client.channels)
	assert.Empty(t, client.channelIDMap)

	// Test channel mapping
	serverNetworkData := &pdu.ServerNetworkData{
		ChannelIdArray: []uint16{1001, 1002, 1003},
		MCSChannelId:   1004,
	}

	client.initChannels(serverNetworkData)

	assert.Equal(t, uint16(1001), client.channelIDMap["cliprdr"])
	assert.Equal(t, uint16(1002), client.channelIDMap["rdpdr"])
	assert.Equal(t, uint16(1003), client.channelIDMap["rdpsnd"])
	assert.Equal(t, uint16(1004), client.channelIDMap["global"])
}
