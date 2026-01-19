package rdp

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient_getServerName_AllBranches(t *testing.T) {
	tests := []struct {
		name           string
		setupClient    func() *Client
		expected       string
	}{
		{
			name: "nil connection",
			setupClient: func() *Client {
				return &Client{conn: nil}
			},
			expected: "",
		},
		{
			name: "empty remote address",
			setupClient: func() *Client {
				return &Client{conn: &tlsExtTestMockConn{remoteAddr: ""}}
			},
			expected: "",
		},
		{
			name: "valid hostname",
			setupClient: func() *Client {
				return &Client{conn: &tlsExtTestMockConn{remoteAddr: "myserver.example.com:3389"}}
			},
			expected: "myserver.example.com",
		},
		{
			name: "IP address",
			setupClient: func() *Client {
				return &Client{conn: &tlsExtTestMockConn{remoteAddr: "192.168.1.100:3389"}}
			},
			expected: "",
		},
		{
			name: "IPv6 address",
			setupClient: func() *Client {
				return &Client{conn: &tlsExtTestMockConn{remoteAddr: "[::1]:3389"}}
			},
			expected: "",
		},
		{
			name: "localhost",
			setupClient: func() *Client {
				return &Client{conn: &tlsExtTestMockConn{remoteAddr: "localhost:3389"}}
			},
			expected: "localhost",
		},
		{
			name: "address without port (fallback)",
			setupClient: func() *Client {
				return &Client{conn: &tlsExtTestMockConn{remoteAddr: "myserver"}}
			},
			expected: "myserver",
		},
		{
			name: "whitespace hostname",
			setupClient: func() *Client {
				return &Client{conn: &tlsExtTestMockConn{remoteAddr: "  myserver.com  :3389"}}
			},
			expected: "myserver.com", // trimmed then hostname check passes
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			result := client.getServerName()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_getMinTLSVersion_AllVersions(t *testing.T) {
	client := &Client{}

	versions := map[string]uint16{
		"1.0":     tls.VersionTLS10,
		"1.1":     tls.VersionTLS11,
		"1.2":     tls.VersionTLS12,
		"1.3":     tls.VersionTLS13,
		"":        tls.VersionTLS12, // default
		"invalid": tls.VersionTLS12, // default
		"2.0":     tls.VersionTLS12, // default
		"0.9":     tls.VersionTLS12, // default
		"1.4":     tls.VersionTLS12, // default (doesn't exist)
	}

	for input, expected := range versions {
		t.Run(input, func(t *testing.T) {
			result := client.getMinTLSVersion(input)
			assert.Equal(t, expected, result)
		})
	}
}

// tlsExtTestMockConn for extended TLS tests
type tlsExtTestMockConn struct {
	net.Conn
	remoteAddr string
}

func (m *tlsExtTestMockConn) Read(b []byte) (n int, err error)  { return 0, nil }
func (m *tlsExtTestMockConn) Write(b []byte) (n int, err error) { return len(b), nil }
func (m *tlsExtTestMockConn) Close() error                      { return nil }
func (m *tlsExtTestMockConn) LocalAddr() net.Addr               { return &tlsExtTestMockAddr{"localhost:8080"} }
func (m *tlsExtTestMockConn) RemoteAddr() net.Addr              { return &tlsExtTestMockAddr{m.remoteAddr} }
func (m *tlsExtTestMockConn) SetDeadline(t time.Time) error     { return nil }
func (m *tlsExtTestMockConn) SetReadDeadline(t time.Time) error { return nil }
func (m *tlsExtTestMockConn) SetWriteDeadline(t time.Time) error { return nil }

type tlsExtTestMockAddr struct{ address string }

func (m *tlsExtTestMockAddr) Network() string { return "tcp" }
func (m *tlsExtTestMockAddr) String() string  { return m.address }
