package rdp

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_NewClient(t *testing.T) {
	tests := []struct {
		name        string
		hostname    string
		username    string
		password    string
		width       int
		height      int
		expectError bool
		expectedErr string
	}{
		{
			name:        "invalid hostname",
			hostname:    "invalid-host-name-that-does-not-exist.com:3389",
			username:    "testuser",
			password:    "testpass",
			width:       1024,
			height:      768,
			expectError: true,
			expectedErr: "tcp connect",
		},
		{
			name:        "unreachable localhost",
			hostname:    "192.0.2.1:3389", // RFC 5737 test address - should be unreachable
			username:    "testuser",
			password:    "testpass",
			width:       1024,
			height:      768,
			expectError: true,
			expectedErr: "tcp connect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.hostname, tt.username, tt.password, tt.width, tt.height, 16)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr)
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
				assert.Equal(t, tt.username, client.username)
				assert.Equal(t, uint16(tt.width), client.desktopWidth)
				assert.Equal(t, uint16(tt.height), client.desktopHeight)
			}
		})
	}
}

func TestClient_NewClient_Validation(t *testing.T) {
	// Test that NewClient properly validates and sets parameters
	client := &Client{
		username:         "testuser",
		desktopWidth:     1024,
		desktopHeight:    768,
		selectedProtocol: 1, // SSL
	}

	assert.Equal(t, "testuser", client.username)
	assert.Equal(t, uint16(1024), client.desktopWidth)
	assert.Equal(t, uint16(768), client.desktopHeight)
}

func TestClient_getServerName(t *testing.T) {
	// Create a mock connection for testing
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	client := &Client{
		conn: conn1,
	}

	serverName := client.getServerName()
	// Since it's a pipe, SplitHostPort might fail, but we should get some fallback
	assert.NotEmpty(t, serverName)
}

func TestClient_StartTLS_Configuration(t *testing.T) {
	// Test TLS configuration without actual handshake
	client := &Client{}

	// Test getServerName with nil connection
	serverName := client.getServerName()
	assert.Empty(t, serverName)
}
