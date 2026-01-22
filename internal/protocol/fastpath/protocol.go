// Package fastpath implements the RDP Fast-Path protocol as specified in MS-RDPBCGR.
// Fast-Path provides optimized encoding for input and output PDUs.
package fastpath

import (
	"io"
)

type Protocol struct {
	conn io.ReadWriter

	updatePDUData []byte
}

func New(conn io.ReadWriter) *Protocol {
	return &Protocol{
		conn: conn,

		updatePDUData: make([]byte, 64*1024),
	}
}
