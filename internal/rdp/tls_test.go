package rdp

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient_getMinTLSVersion(t *testing.T) {
	client := &Client{}

	tests := []struct {
		name     string
		version  string
		expected uint16
	}{
		{"TLS 1.0", "1.0", tls.VersionTLS10},
		{"TLS 1.1", "1.1", tls.VersionTLS11},
		{"TLS 1.2", "1.2", tls.VersionTLS12},
		{"TLS 1.3", "1.3", tls.VersionTLS13},
		{"empty string defaults to 1.2", "", tls.VersionTLS12},
		{"invalid version defaults to 1.2", "invalid", tls.VersionTLS12},
		{"version 2.0 defaults to 1.2", "2.0", tls.VersionTLS12},
		{"version 0.9 defaults to 1.2", "0.9", tls.VersionTLS12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.getMinTLSVersion(tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestClient_getServerName_NilConnection(t *testing.T) {
	client := &Client{conn: nil}
	serverName := client.getServerName()
	assert.Empty(t, serverName)
}

func TestClient_getServerName_WithMockConnection(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		expectedResult string
	}{
		{
			name:           "valid hostname with port",
			remoteAddr:     "server.example.com:3389",
			expectedResult: "server.example.com",
		},
		{
			name:           "IP address returns empty",
			remoteAddr:     "192.168.1.1:3389",
			expectedResult: "",
		},
		{
			name:           "IPv6 address returns empty",
			remoteAddr:     "[::1]:3389",
			expectedResult: "",
		},
		{
			name:           "localhost IP returns empty",
			remoteAddr:     "127.0.0.1:3389",
			expectedResult: "",
		},
		{
			name:           "short hostname",
			remoteAddr:     "host:3389",
			expectedResult: "host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &tlsTestMockConn{remoteAddr: tt.remoteAddr}
			client := &Client{conn: conn}
			result := client.getServerName()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestClient_getServerName_LongHostname(t *testing.T) {
	// Create a hostname longer than 253 characters
	longHostname := ""
	for i := 0; i < 260; i++ {
		longHostname += "a"
	}
	longHostname += ":3389"

	conn := &tlsTestMockConn{remoteAddr: longHostname}
	client := &Client{conn: conn}
	result := client.getServerName()
	assert.Empty(t, result, "hostnames longer than 253 chars should return empty")
}

func TestClient_getServerName_EmptyRemoteAddr(t *testing.T) {
	conn := &tlsTestMockConn{remoteAddr: ""}
	client := &Client{conn: conn}
	result := client.getServerName()
	assert.Empty(t, result)
}

// tlsTestMockConn is a mock net.Conn for TLS testing
type tlsTestMockConn struct {
	net.Conn
	remoteAddr string
}

func (m *tlsTestMockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (m *tlsTestMockConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (m *tlsTestMockConn) Close() error {
	return nil
}

func (m *tlsTestMockConn) LocalAddr() net.Addr {
	return &tlsTestMockAddr{"localhost:8080"}
}

func (m *tlsTestMockConn) RemoteAddr() net.Addr {
	return &tlsTestMockAddr{m.remoteAddr}
}

func (m *tlsTestMockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *tlsTestMockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *tlsTestMockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type tlsTestMockAddr struct {
	address string
}

func (m *tlsTestMockAddr) Network() string {
	return "tcp"
}

func (m *tlsTestMockAddr) String() string {
	return m.address
}

func TestClient_SetTLSConfig(t *testing.T) {
	client := &Client{}

	// Test default values
	assert.False(t, client.skipTLSValidation)
	assert.Empty(t, client.tlsServerName)

	// Test setting values
	client.SetTLSConfig(true, "test-server.com")
	assert.True(t, client.skipTLSValidation)
	assert.Equal(t, "test-server.com", client.tlsServerName)

	// Test changing values
	client.SetTLSConfig(false, "another-server.com")
	assert.False(t, client.skipTLSValidation)
	assert.Equal(t, "another-server.com", client.tlsServerName)
}
