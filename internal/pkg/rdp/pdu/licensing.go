package pdu

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/kulaginds/rdp-html5/internal/pkg/rdp/headers"
)

type LicensingBinaryBlob struct {
	BlobType uint16
	BlobLen  uint16
	BlobData []byte
}

func (b *LicensingBinaryBlob) Deserialize(wire io.Reader) error {
	binary.Read(wire, binary.LittleEndian, &b.BlobType)
	binary.Read(wire, binary.LittleEndian, &b.BlobLen)

	if b.BlobLen == 0 {
		return nil
	}

	b.BlobData = make([]byte, b.BlobLen)

	if _, err := wire.Read(b.BlobData); err != nil {
		return err
	}

	return nil
}

type LicensingErrorMessage struct {
	ErrorCode       uint32
	StateTransition uint32
	ErrorInfo       LicensingBinaryBlob
}

func (m *LicensingErrorMessage) Deserialize(wire io.Reader) error {
	binary.Read(wire, binary.LittleEndian, &m.ErrorCode)
	binary.Read(wire, binary.LittleEndian, &m.StateTransition)

	return m.ErrorInfo.Deserialize(wire)
}

type LicensingPreamble struct {
	MsgType uint8
	Flags   uint8
	MsgSize uint16
}

func (p *LicensingPreamble) Deserialize(wire io.Reader) error {
	binary.Read(wire, binary.LittleEndian, &p.MsgType)
	binary.Read(wire, binary.LittleEndian, &p.Flags)
	binary.Read(wire, binary.LittleEndian, &p.MsgSize)

	return nil
}

type ServerLicenseError struct {
	Preamble           LicensingPreamble
	ValidClientMessage LicensingErrorMessage
}

// Deserialize parses the server license response.
// Note: XRDP sends security header even with TLS, so we always expect it.
func (pdu *ServerLicenseError) Deserialize(wire io.Reader, useEnhancedSecurity bool) error {
	// Always expect security header for XRDP compatibility.
	// XRDP sends SEC_LICENSE_PKT | SEC_LICENSE_ENCRYPT_CS (0x0280) even with TLS.
	securityFlag, err := headers.UnwrapSecurityFlag(wire)
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
