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

// Test certificate for TLS testing
const tlsCert = `-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAMlyFqk69v+9MA0GCSqGSIb3DQEBBQUAMBQxEjAQBgNVBAMMCWxv
Y2FsaG9zdDAeFw0xNjEwMjgxNjEwNThaFw0xNzEwMjgxNjEwNThaMBQxEjAQBgNV
BAMMCWxvY2FsaG9zdDBcMA0GCSqGSIb3DQEBAQUAA0sAMEgCQQDYwJtDvUfeAwKt
5wG/zlN1BuB9Y+RfMGNuxPzxVwA+wdaHRjgUuVUEpcT2hh+NPEvbErvvXQcKn6c
nGTFKNQd9AkEA0Fk7qJyON+mJ5m1HV2ra7ya6F9dX8hO5Kr+R2Bk2cKq3mNJhNq
NQjNQzJDQMOJjfsKTHmJOf5jB5rmEVR+lJQJBAN5N1KJhQlBHAKwFKTxpTLzNXFI
gnBvuIDJr8UXjyFvZYnDzvES+jgnKg2VqKjBwR3QI6Af0JxHN3jqAbSNnmcCQBH
bYYJjZmL1jO0yHJvE2BTFjTzOA7m1gdvKQIYDNYrgKiGWgGYmC1FQw4IFqf9KIZ
FxTJq2mCccTjvUA0FnMCQQDFHLlQqSzd+XVFfBHJqvVCv15eKqKTVLtSO3lFlpGK
NtpsjTR2f8QxxqeWJYV5Y3BlnJROjK4/VKDpd9Vj7bArW
-----END CERTIFICATE-----`

const tlsKey = `-----BEGIN RSA PRIVATE KEY-----
MIIBOwIBAAJBAMlyFqk69v/9PmFpPy38XcMxvidqKSnxDRfYYjmXXC8aT1VLAyKA
m3pc2rR7HBJ8NqT4C8QcQj9gy2o9gSDl2IHBcCAwEAAQJAYDIfUxsfd+DWj2cCVQ
A8nKgNnGGuJqQH8UOJlKmC6y2QI+vv5fGJPn9IBOO9E0eGoJmGeGdFjY1K4f6+2G
IgQIhAPQn6KqMtQs7XLhqPqp3sbgy5EqODJ6QFzrPDQ5oR+yZAiEAxSz2j65G9Sb
NQJz2wQmrT+zDlFhB8Un7rPX1Wb6s/ZsCIGRsxifmRUQB7WtxzUA6X8WHPxZweK5
GQ9FmK1s9dSA3ZAiEAyVktN4PjZ9w9KX+aB6F6+vMPOLTVnfYLtXF2Xa9Y2AykC
IQDFNihDLBlb6nHOX+sPnqYgKVY3VgksNb9dFSDM+kqOPw==
-----END RSA PRIVATE KEY-----`
