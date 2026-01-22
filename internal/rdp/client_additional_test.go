package rdp

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewClient_Success(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = ln.Close() }()

	accepted := make(chan net.Conn, 1)
	go func() {
		c, aerr := ln.Accept()
		if aerr == nil {
			accepted <- c
		}
	}()

	client, err := NewClient(ln.Addr().String(), "user", "pass", 1024, 768, 16)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer func() { _ = client.Close() }()

	// Ensure we did accept a connection on the server side.
	conn := <-accepted
	_ = conn.Close()
}

func TestNewClient_InvalidPort(t *testing.T) {
	_, err := NewClient("127.0.0.1:0", "user", "pass", 1024, 768, 16)
	require.Error(t, err)
}
