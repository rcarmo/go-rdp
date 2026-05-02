// Package codec exposes RDP codec/security helpers from go-rdp.
package codec

import (
	"io"

	internalcodec "github.com/rcarmo/go-rdp/internal/codec"
)

func WrapSecurityFlag(flag uint16, data []byte) []byte {
	return internalcodec.WrapSecurityFlag(flag, data)
}
func UnwrapSecurityFlag(wire io.Reader) (uint16, error) {
	return internalcodec.UnwrapSecurityFlag(wire)
}
