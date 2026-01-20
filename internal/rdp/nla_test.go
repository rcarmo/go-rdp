package rdp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_parseDomainUser(t *testing.T) {
	tests := []struct {
		name           string
		username       string
		clientDomain   string
		expectedDomain string
		expectedUser   string
	}{
		{
			name:           "DOMAIN\\user format",
			username:       "MYDOMAIN\\testuser",
			clientDomain:   "",
			expectedDomain: "MYDOMAIN",
			expectedUser:   "testuser",
		},
		{
			name:           "user@domain format",
			username:       "testuser@example.com",
			clientDomain:   "",
			expectedDomain: "example.com",
			expectedUser:   "testuser",
		},
		{
			name:           "plain username with client domain",
			username:       "testuser",
			clientDomain:   "WORKGROUP",
			expectedDomain: "WORKGROUP",
			expectedUser:   "testuser",
		},
		{
			name:           "plain username without domain",
			username:       "localuser",
			clientDomain:   "",
			expectedDomain: "",
			expectedUser:   "localuser",
		},
		{
			name:           "backslash at start",
			username:       "\\user",
			clientDomain:   "",
			expectedDomain: "",
			expectedUser:   "user",
		},
		{
			name:           "backslash at end",
			username:       "DOMAIN\\",
			clientDomain:   "",
			expectedDomain: "DOMAIN",
			expectedUser:   "",
		},
		{
			name:           "multiple backslashes",
			username:       "A\\B\\C",
			clientDomain:   "",
			expectedDomain: "A",
			expectedUser:   "B\\C",
		},
		{
			name:           "at sign at start",
			username:       "@domain.com",
			clientDomain:   "",
			expectedDomain: "domain.com",
			expectedUser:   "",
		},
		{
			name:           "at sign at end",
			username:       "user@",
			clientDomain:   "",
			expectedDomain: "",
			expectedUser:   "user",
		},
		{
			name:           "multiple at signs",
			username:       "user@sub@domain.com",
			clientDomain:   "",
			expectedDomain: "sub@domain.com",
			expectedUser:   "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				username: tt.username,
				domain:   tt.clientDomain,
			}

			domain, user := client.parseDomainUser()

			assert.Equal(t, tt.expectedDomain, domain)
			assert.Equal(t, tt.expectedUser, user)
		})
	}
}

func TestParseASN1Length(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		expectedLength int
		expectedBytes  int
	}{
		{
			name:           "empty data",
			data:           []byte{},
			expectedLength: 0,
			expectedBytes:  0,
		},
		{
			name:           "short form - zero",
			data:           []byte{0x00},
			expectedLength: 0,
			expectedBytes:  1,
		},
		{
			name:           "short form - 127",
			data:           []byte{0x7F},
			expectedLength: 127,
			expectedBytes:  1,
		},
		{
			name:           "long form - 1 byte length",
			data:           []byte{0x81, 0xFF},
			expectedLength: 255,
			expectedBytes:  2,
		},
		{
			name:           "long form - 2 byte length",
			data:           []byte{0x82, 0x01, 0x00},
			expectedLength: 256,
			expectedBytes:  3,
		},
		{
			name:           "long form - 2 byte length bigger",
			data:           []byte{0x82, 0x04, 0x00},
			expectedLength: 1024,
			expectedBytes:  3,
		},
		{
			name:           "long form - 3 byte length",
			data:           []byte{0x83, 0x01, 0x00, 0x00},
			expectedLength: 65536,
			expectedBytes:  4,
		},
		{
			name:           "long form - 4 byte length",
			data:           []byte{0x84, 0x01, 0x00, 0x00, 0x00},
			expectedLength: 16777216,
			expectedBytes:  5,
		},
		{
			name:           "invalid - numBytes is 0",
			data:           []byte{0x80},
			expectedLength: 0,
			expectedBytes:  1,
		},
		{
			name:           "invalid - numBytes > 4",
			data:           []byte{0x85, 0x00, 0x00, 0x00, 0x00, 0x00},
			expectedLength: 0,
			expectedBytes:  1,
		},
		{
			name:           "invalid - not enough bytes",
			data:           []byte{0x82, 0x01}, // needs 2 bytes but only 1
			expectedLength: 0,
			expectedBytes:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length, bytes := parseASN1Length(tt.data)
			require.Equal(t, tt.expectedLength, length, "length mismatch")
			require.Equal(t, tt.expectedBytes, bytes, "bytes consumed mismatch")
		})
	}
}
