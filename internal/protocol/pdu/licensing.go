// Package pdu implements RDP Protocol Data Units as defined in MS-RDPBCGR.
package pdu

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/rcarmo/rdp-html5/internal/codec"
)

// LicensingBinaryBlob represents a LICENSE_BINARY_BLOB structure (MS-RDPELE 2.2.2.4).
type LicensingBinaryBlob struct {
	BlobType uint16
	BlobLen  uint16
	BlobData []byte
}

// Deserialize reads a LICENSE_BINARY_BLOB from wire.
func (b *LicensingBinaryBlob) Deserialize(wire io.Reader) error {
	if err := binary.Read(wire, binary.LittleEndian, &b.BlobType); err != nil {
		return err
	}
	if err := binary.Read(wire, binary.LittleEndian, &b.BlobLen); err != nil {
		return err
	}

	if b.BlobLen == 0 {
		return nil
	}

	b.BlobData = make([]byte, b.BlobLen)

	if _, err := wire.Read(b.BlobData); err != nil {
		return err
	}

	return nil
}

// LicensingErrorMessage represents a LICENSE_ERROR_MESSAGE structure (MS-RDPELE 2.2.1.12).
type LicensingErrorMessage struct {
	ErrorCode       uint32
	StateTransition uint32
	ErrorInfo       LicensingBinaryBlob
}

// Deserialize reads a LICENSE_ERROR_MESSAGE from wire.
func (m *LicensingErrorMessage) Deserialize(wire io.Reader) error {
	if err := binary.Read(wire, binary.LittleEndian, &m.ErrorCode); err != nil {
		return err
	}
	if err := binary.Read(wire, binary.LittleEndian, &m.StateTransition); err != nil {
		return err
	}

	return m.ErrorInfo.Deserialize(wire)
}

// LicensingPreamble represents a LICENSE_PREAMBLE structure (MS-RDPELE 2.2.2.1).
type LicensingPreamble struct {
	MsgType uint8
	Flags   uint8
	MsgSize uint16
}

// Deserialize reads a LICENSE_PREAMBLE from wire.
func (p *LicensingPreamble) Deserialize(wire io.Reader) error {
	if err := binary.Read(wire, binary.LittleEndian, &p.MsgType); err != nil {
		return err
	}
	if err := binary.Read(wire, binary.LittleEndian, &p.Flags); err != nil {
		return err
	}
	return binary.Read(wire, binary.LittleEndian, &p.MsgSize)
}

// ServerLicenseError represents a Server License Error PDU (MS-RDPBCGR 2.2.1.12).
type ServerLicenseError struct {
	Preamble           LicensingPreamble
	ValidClientMessage LicensingErrorMessage
}

// Deserialize parses the server license response.
// Note: XRDP sends security header even with TLS, so we always expect it.
func (pdu *ServerLicenseError) Deserialize(wire io.Reader, useEnhancedSecurity bool) error {
	// Always expect security header for XRDP compatibility.
	// XRDP sends SEC_LICENSE_PKT | SEC_LICENSE_ENCRYPT_CS (0x0280) even with TLS.
	securityFlag, err := codec.UnwrapSecurityFlag(wire)
	if err != nil {
		return err
	}

	// SEC_LICENSE_PKT = 0x0080, may be combined with SEC_LICENSE_ENCRYPT_CS = 0x0200
	if securityFlag&0x0080 == 0 { // SEC_LICENSE_PKT
		return errors.New("bad license header")
	}

	err = pdu.Preamble.Deserialize(wire)
	if err != nil {
		return err
	}

	err = pdu.ValidClientMessage.Deserialize(wire)
	if err != nil {
		return err
	}

	return nil
}
