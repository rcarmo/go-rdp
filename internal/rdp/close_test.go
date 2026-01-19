package rdp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_Close_WithRemoteApp(t *testing.T) {
	// Create a client with a RemoteApp
	client := &Client{
		remoteApp: &RemoteApp{
			App:        "test.exe",
			WorkingDir: "C:\\",
			Args:       "",
		},
		railState: RailStateExecuteApp,
		conn:      &closeTestMockConn{},
	}

	err := client.Close()

	assert.NoError(t, err)
	assert.Equal(t, RailStateUninitialized, client.railState)
}

func TestClient_Close_WithoutRemoteApp(t *testing.T) {
	// Create a client without a RemoteApp
	client := &Client{
		remoteApp: nil,
		railState: RailStateWaitForData, // Should not change
		conn:      &closeTestMockConn{},
	}

	err := client.Close()

	assert.NoError(t, err)
	// railState should not change when remoteApp is nil
	assert.Equal(t, RailStateWaitForData, client.railState)
}

// closeTestMockConn is a mock connection for testing Close
type closeTestMockConn struct {
	mockConn
}
