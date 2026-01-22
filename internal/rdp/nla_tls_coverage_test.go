package rdp

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSendFrameAcknowledge_Success tests the frame acknowledge functionality
func TestSendFrameAcknowledge_Success(t *testing.T) {
	tests := []struct {
		name      string
		frameID   uint32
		shareID   uint32
		userID    uint16
		sendErr   error
		expectErr bool
	}{
		{
			name:      "successful ack",
			frameID:   12345,
			shareID:   0x12345678,
			userID:    1001,
			expectErr: false,
		},
		{
			name:      "frame id zero",
			frameID:   0,
			shareID:   0x12345678,
			userID:    1001,
			expectErr: false,
		},
		{
			name:      "max frame id",
			frameID:   0xFFFFFFFF,
			shareID:   0xFFFFFFFF,
			userID:    65535,
			expectErr: false,
		},
		{
			name:      "send error",
			frameID:   1,
			shareID:   1,
			userID:    1001,
			sendErr:   errors.New("send failed"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sentData []byte
			mockMCS := &MockMCSLayer{
				SendFunc: func(userID, channelID uint16, data []byte) error {
					sentData = data
					return tt.sendErr
				},
			}

			client := &Client{
				shareID:      tt.shareID,
				userID:       tt.userID,
				mcsLayer:     mockMCS,
				channelIDMap: map[string]uint16{"global": 1003},
			}

			err := client.SendFrameAcknowledge(tt.frameID)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, sentData)
			}
		})
	}
}

// TestGetMinTLSVersion_Parsing tests TLS version parsing
func TestGetMinTLSVersion_Parsing(t *testing.T) {
	tests := []struct {
		version string
		expect  uint16
	}{
		{"1.0", tls.VersionTLS10},
		{"1.1", tls.VersionTLS11},
		{"1.2", tls.VersionTLS12},
		{"1.3", tls.VersionTLS13},
		{"invalid", tls.VersionTLS12}, // default
		{"", tls.VersionTLS12},        // default
		{"2.0", tls.VersionTLS12},     // unknown version
	}

	client := &Client{}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := client.getMinTLSVersion(tt.version)
			assert.Equal(t, tt.expect, result)
		})
	}
}

// TestGetServerName_Extraction tests server name extraction
func TestGetServerName_Extraction(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		expectName string
	}{
		{
			name:       "IP address returns empty",
			remoteAddr: "192.168.1.100:3389",
			expectName: "",
		},
		{
			name:       "IPv6 address",
			remoteAddr: "[::1]:3389",
			expectName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &testNetConn{remoteAddr: tt.remoteAddr}
			client := &Client{conn: conn}
			result := client.getServerName()
			assert.Equal(t, tt.expectName, result)
		})
	}
}

// TestGetServerName_NilConn tests server name with nil connection
func TestGetServerName_NilConn(t *testing.T) {
	client := &Client{conn: nil}
	result := client.getServerName()
	assert.Empty(t, result)
}

// TestReadNLAMessage_SizeLimits tests NLA message reading with size limits
func TestReadNLAMessage_SizeLimits(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		maxSize   int
		expectErr bool
	}{
		{
			name:      "small message",
			data:      []byte{0x30, 0x05, 0x01, 0x02, 0x03},
			maxSize:   1024,
			expectErr: false,
		},
		{
			name:      "message at limit",
			data:      bytes.Repeat([]byte{0x01}, 100),
			maxSize:   100,
			expectErr: false,
		},
		{
			name:      "message exceeds limit",
			data:      bytes.Repeat([]byte{0x01}, 200),
			maxSize:   100,
			expectErr: true,
		},
		{
			name:      "empty message",
			data:      []byte{},
			maxSize:   1024,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			result, err := readNLAMessage(reader, tt.maxSize)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if len(tt.data) > 0 {
					assert.NotEmpty(t, result)
				}
			}
		})
	}
}

// TestParseDomainUser_Formats tests domain and user parsing
func TestParseDomainUser_Formats(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		expectDomain string
		expectUser   string
	}{
		{
			name:         "simple username",
			username:     "user",
			expectDomain: "",
			expectUser:   "user",
		},
		{
			name:         "domain backslash format",
			username:     "DOMAIN\\user",
			expectDomain: "DOMAIN",
			expectUser:   "user",
		},
		{
			name:         "UPN format",
			username:     "user@domain.com",
			expectDomain: "domain.com",
			expectUser:   "user",
		},
		{
			name:         "empty username",
			username:     "",
			expectDomain: "",
			expectUser:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				domain:   "",
				username: tt.username,
			}

			domain, user := client.parseDomainUser()
			assert.Equal(t, tt.expectDomain, domain)
			assert.Equal(t, tt.expectUser, user)
		})
	}
}

// TestParseDomainUser_ExplicitDomain tests parsing with explicit domain set
func TestParseDomainUser_ExplicitDomain(t *testing.T) {
	client := &Client{
		domain:   "EXPLICIT",
		username: "user",
	}

	domain, user := client.parseDomainUser()
	assert.Equal(t, "EXPLICIT", domain)
	assert.Equal(t, "user", user)
}

// TestParseASN1Length_Scenarios tests ASN.1 length parsing
func TestParseASN1Length_Scenarios(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		expectLen int
		expectOff int
	}{
		{
			name:      "short form",
			data:      []byte{0x05},
			expectLen: 5,
			expectOff: 1,
		},
		{
			name:      "long form one byte",
			data:      []byte{0x81, 0x80},
			expectLen: 128,
			expectOff: 2,
		},
		{
			name:      "long form two bytes",
			data:      []byte{0x82, 0x01, 0x00},
			expectLen: 256,
			expectOff: 3,
		},
		{
			name:      "empty data returns zero",
			data:      []byte{},
			expectLen: 0,
			expectOff: 0,
		},
		{
			name:      "incomplete long form returns zero",
			data:      []byte{0x82, 0x01},
			expectLen: 0,
			expectOff: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length, offset := parseASN1Length(tt.data)
			assert.Equal(t, tt.expectLen, length)
			assert.Equal(t, tt.expectOff, offset)
		})
	}
}

// testNetConn implements net.Conn for testing
type testNetConn struct {
	remoteAddr string
	readData   []byte
	readErr    error
	writeErr   error
	closeErr   error
}

func (m *testNetConn) Read(b []byte) (n int, err error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	n = copy(b, m.readData)
	return n, io.EOF
}

func (m *testNetConn) Write(b []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return len(b), nil
}

func (m *testNetConn) Close() error {
	return m.closeErr
}

func (m *testNetConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

func (m *testNetConn) RemoteAddr() net.Addr {
	host, port, err := net.SplitHostPort(m.remoteAddr)
	if err != nil {
		return &net.TCPAddr{}
	}
	portInt := 3389
	if port != "" {
		// Simple port parsing for tests
		for i := 0; i < len(port); i++ {
			if port[i] >= '0' && port[i] <= '9' {
				portInt = portInt*10 + int(port[i]-'0')
			}
		}
	}
	return &net.TCPAddr{IP: net.ParseIP(host), Port: portInt}
}

func (m *testNetConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *testNetConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *testNetConn) SetWriteDeadline(t time.Time) error {
	return nil
}
