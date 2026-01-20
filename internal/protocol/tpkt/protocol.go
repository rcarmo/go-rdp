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
