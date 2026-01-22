// Package tpkt implements the TPKT transport protocol (RFC 1006) used as
// the base transport layer for RDP connections.
package tpkt

import (
	"io"
)

const (
	headerLen = 4
)

type Protocol struct {
	conn io.ReadWriteCloser
}

func New(conn io.ReadWriteCloser) *Protocol {
	return &Protocol{
		conn: conn,
	}
}
