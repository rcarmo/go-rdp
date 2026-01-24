// Package x224 implements the X.224 connection-oriented transport protocol
// used in the RDP connection sequence for initial negotiation.
package x224

import (
	"io"

	"github.com/rcarmo/go-rdp/internal/protocol/tpkt"
)

// tpktConnection is the interface that wraps tpkt protocol operations
type tpktConnection interface {
	Receive() (io.Reader, error)
	Send(pduData []byte) error
}

// Protocol handles X.224 protocol operations
type Protocol struct {
	tpktConn tpktConnection
}

// New creates a new X.224 protocol handler
func New(tpktConn *tpkt.Protocol) *Protocol {
	return &Protocol{
		tpktConn: tpktConn,
	}
}

// NewWithConn creates a new X.224 protocol handler with an interface (for testing)
func NewWithConn(conn tpktConnection) *Protocol {
	return &Protocol{
		tpktConn: conn,
	}
}
