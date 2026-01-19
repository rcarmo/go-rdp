package rdp

import (
	"net"
	"testing"
	"time"

	"github.com/rcarmo/rdp-html5/internal/protocol/pdu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_getServerName_extended(t *testing.T) {
	tests := []struct {
		name           string
		setupConn      func() net.Conn
		expectedResult string
		expectNonEmpty bool
	}{
		{
			name: "pipe connection returns non-empty",
			setupConn: func() net.Conn {
				// Create a mock connection - pipe returns "pipe" as address
				_, client := net.Pipe()
				return client
			},
			expectNonEmpty: true, // Pipe returns "pipe" which is a valid hostname
		},
		{
			name: "nil connection",
			setupConn: func() net.Conn {
				return nil
			},
			expectNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := tt.setupConn()
			client := &Client{
				conn: conn,
			}
			if conn != nil {
				defer conn.Close()
			}

			result := client.getServerName()

			if tt.expectNonEmpty {
				assert.NotEmpty(t, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestClient_StartTLS_Configuration_extended(t *testing.T) {
	// Skip tests that would hang due to TLS handshake
	// The actual TLS functionality is tested via integration tests
	t.Skip("Skipping TLS tests that require mock TLS server")
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
	client := &Client{
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
	client := &Client{
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
			client := &Client{}
			client.skipChannelJoin = tt.earlyCapabilityFlags&0x8 == 0x8

			client.skipChannelJoin = tt.earlyCapabilityFlags&0x8 == 0x8

			assert.Equal(t, tt.expectedSkip, client.skipChannelJoin)
		})
	}
}

// Integration-style tests for client lifecycle
func TestClient_ConnectionFlow(t *testing.T) {
	// Test client initialization phases
	client := &Client{
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
			client, err := NewClient(tt.hostname, tt.username, tt.password, tt.width, tt.height, 16)

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

	client := &Client{
		remoteApp: app,
	}

	assert.Equal(t, app, client.remoteApp)
}

func TestClient_Channels(t *testing.T) {
	channels := []string{"cliprdr", "rdpdr", "rdpsnd"}
	client := &Client{
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
