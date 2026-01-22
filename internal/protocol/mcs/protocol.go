// Package mcs implements the Multipoint Communication Service (T.125) protocol
// layer for RDP connections as specified in MS-RDPBCGR.
package mcs

import "github.com/rcarmo/rdp-html5/internal/protocol/x224"

type Protocol struct {
	x224Conn x224Conn
}

func New(x224Conn *x224.Protocol) *Protocol {
	return &Protocol{
		x224Conn: x224Conn,
	}
}

// newWithConn creates a Protocol with a custom x224Conn (for testing)
func newWithConn(conn x224Conn) *Protocol {
	return &Protocol{
		x224Conn: conn,
	}
}
