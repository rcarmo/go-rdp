package rdp

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
