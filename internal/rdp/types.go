// Package rdp provides an RDP client implementation for connecting to
// Windows Remote Desktop servers.
package rdp

import (
	"bufio"
)

// ProtocolCode represents the first byte of an RDP message used to determine
// whether the message uses FastPath or X.224 framing.
type ProtocolCode uint8

// IsFastpath returns true if the protocol code indicates a FastPath message.
func (a ProtocolCode) IsFastpath() bool {
	return a&0x3 == 0
}

// IsX224 returns true if the protocol code indicates an X.224 message.
func (a ProtocolCode) IsX224() bool {
	return a == 3
}

func receiveProtocol(bufReader *bufio.Reader) (ProtocolCode, error) {
	action, err := bufReader.ReadByte()
	if err != nil {
		return 0, err
	}

	err = bufReader.UnreadByte()
	if err != nil {
		return 0, err
	}

	return ProtocolCode(action), nil
}
