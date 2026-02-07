// Package pdu implements RDP Protocol Data Units for the connection sequence,
// capabilities exchange, input/output events, and licensing as specified in MS-RDPBCGR.
package pdu

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Type represents the PDU type field in share control headers (MS-RDPBCGR 2.2.8.1.1.1.1).
type Type uint16

const (
	// TypeDemandActive PDUTYPE_DEMANDACTIVEPDU
	TypeDemandActive Type = 0x11

	// TypeConfirmActive PDUTYPE_CONFIRMACTIVEPDU
	TypeConfirmActive Type = 0x13

	// TypeDeactivateAll PDUTYPE_DEACTIVATEALLPDU
	TypeDeactivateAll Type = 0x16

	// TypeData PDUTYPE_DATAPDU
	TypeData Type = 0x17
)

// IsDemandActive returns true if the PDU type is Demand Active.
func (t Type) IsDemandActive() bool {
	return t == TypeDemandActive
}

// IsConfirmActive returns true if the PDU type is Confirm Active.
func (t Type) IsConfirmActive() bool {
	return t == TypeConfirmActive
}

// IsDeactivateAll returns true if the PDU type is Deactivate All.
func (t Type) IsDeactivateAll() bool {
	return t == TypeDeactivateAll
}

// IsData returns true if the PDU type is Data.
func (t Type) IsData() bool {
	return t == TypeData
}

// ShareControlHeader represents the TS_SHARECONTROLHEADER structure (MS-RDPBCGR 2.2.8.1.1.1.1).
type ShareControlHeader struct {
	TotalLength uint16
	PDUType     Type
	PDUSource   uint16
}

func newShareControlHeader(pduType Type, pduSource uint16) *ShareControlHeader {
	return &ShareControlHeader{
		PDUType:   pduType,
		PDUSource: pduSource,
	}
}

// Serialize encodes the header to wire format.
func (header *ShareControlHeader) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, header.TotalLength)
	_ = binary.Write(buf, binary.LittleEndian, uint16(header.PDUType))
	_ = binary.Write(buf, binary.LittleEndian, header.PDUSource)

	return buf.Bytes()
}

// Deserialize decodes the header from wire format.
func (header *ShareControlHeader) Deserialize(wire io.Reader) error {
	if err := binary.Read(wire, binary.LittleEndian, &header.TotalLength); err != nil {
		return err
	}
	if err := binary.Read(wire, binary.LittleEndian, &header.PDUType); err != nil {
		return err
	}
	return binary.Read(wire, binary.LittleEndian, &header.PDUSource)
}

// Type2 represents the PDU type 2 field in share data headers (MS-RDPBCGR 2.2.8.1.1.1.2).
type Type2 uint8

const (
	// Type2Update PDUTYPE2_UPDATE
	Type2Update Type2 = 0x02

	// Type2Control PDUTYPE2_CONTROL
	Type2Control Type2 = 0x14

	// Type2Pointer PDUTYPE2_POINTER
	Type2Pointer Type2 = 0x1B

	// Type2Input PDUTYPE2_INPUT
	Type2Input Type2 = 0x1C

	// Type2Synchronize PDUTYPE2_SYNCHRONIZE
	Type2Synchronize Type2 = 0x1F

	// Type2Fontlist PDUTYPE2_FONTLIST
	Type2Fontlist Type2 = 0x27

	// Type2Fontmap PDUTYPE2_FONTMAP
	Type2Fontmap Type2 = 0x28

	// Type2ErrorInfo PDUTYPE2_SET_ERROR_INFO_PDU
	Type2ErrorInfo Type2 = 0x2f

	// Type2SaveSessionInfo PDUTYPE2_SAVE_SESSION_INFO
	Type2SaveSessionInfo Type2 = 0x26
)

// IsUpdate returns true if the PDU type 2 is Update.
func (t Type2) IsUpdate() bool {
	return t == Type2Update
}

// IsControl returns true if the PDU type 2 is Control.
func (t Type2) IsControl() bool {
	return t == Type2Control
}

// IsPointer returns true if the PDU type 2 is Pointer.
func (t Type2) IsPointer() bool {
	return t == Type2Pointer
}

// IsInput returns true if the PDU type 2 is Input.
func (t Type2) IsInput() bool {
	return t == Type2Input
}

// IsSynchronize returns true if the PDU type 2 is Synchronize.
func (t Type2) IsSynchronize() bool {
	return t == Type2Synchronize
}

// IsFontlist returns true if the PDU type 2 is Font List.
func (t Type2) IsFontlist() bool {
	return t == Type2Fontlist
}

// IsErrorInfo returns true if the PDU type 2 is Error Info.
func (t Type2) IsErrorInfo() bool {
	return t == Type2ErrorInfo
}

// IsFontmap returns true if the PDU type 2 is Font Map.
func (t Type2) IsFontmap() bool {
	return t == Type2Fontmap
}

// IsSaveSessionInfo returns true if the PDU type 2 is Save Session Info.
func (t Type2) IsSaveSessionInfo() bool {
	return t == Type2SaveSessionInfo
}

// ShareDataHeader represents the TS_SHAREDATAHEADER structure (MS-RDPBCGR 2.2.8.1.1.1.2).
type ShareDataHeader struct {
	ShareControlHeader ShareControlHeader
	ShareID            uint32
	StreamID           uint8
	UncompressedLength uint16
	PDUType2           Type2
	CompressedType     uint8
	CompressedLength   uint16
}

func newShareDataHeader(shareID uint32, pduSource uint16, pduType Type, pduType2 Type2) *ShareDataHeader {
	return &ShareDataHeader{
		ShareControlHeader: *newShareControlHeader(pduType, pduSource),
		ShareID:            shareID,
		StreamID:           0x01, // STREAM_LOW
		PDUType2:           pduType2,
	}
}

// Serialize encodes the header to wire format.
func (header *ShareDataHeader) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.Write(header.ShareControlHeader.Serialize())
	_ = binary.Write(buf, binary.LittleEndian, header.ShareID)
	_ = binary.Write(buf, binary.LittleEndian, uint8(0)) // padding
	_ = binary.Write(buf, binary.LittleEndian, header.StreamID)
	_ = binary.Write(buf, binary.LittleEndian, header.UncompressedLength)
	_ = binary.Write(buf, binary.LittleEndian, uint8(header.PDUType2))
	_ = binary.Write(buf, binary.LittleEndian, header.CompressedType)
	_ = binary.Write(buf, binary.LittleEndian, header.CompressedLength)

	return buf.Bytes()
}

// Deserialize decodes the header from wire format.
func (header *ShareDataHeader) Deserialize(wire io.Reader) error {
	var (
		padding uint8
		err     error
	)

	if err = header.ShareControlHeader.Deserialize(wire); err != nil {
		return err
	}

	if header.ShareControlHeader.PDUType.IsDeactivateAll() {
		return ErrDeactivateAll
	}

	err = binary.Read(wire, binary.LittleEndian, &header.ShareID)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &header.StreamID)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &header.UncompressedLength)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &header.PDUType2)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &header.CompressedType)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &header.CompressedLength)
	if err != nil {
		return err
	}

	return nil
}

// Data represents a share data PDU containing one of several data types (MS-RDPBCGR 2.2.8.1.1.1).
type Data struct {
	ShareDataHeader    ShareDataHeader
	SynchronizePDUData *SynchronizePDUData
	ControlPDUData     *ControlPDUData
	FontListPDUData    *FontListPDUData
	FontMapPDUData     *FontMapPDUData
	ErrorInfoPDUData   *ErrorInfoPDUData
}

// Serialize encodes the PDU to wire format.
func (pdu *Data) Serialize() []byte {
	var data []byte

	switch {
	case pdu.ShareDataHeader.PDUType2.IsSynchronize():
		data = pdu.SynchronizePDUData.Serialize()
	case pdu.ShareDataHeader.PDUType2.IsControl():
		data = pdu.ControlPDUData.Serialize()
	case pdu.ShareDataHeader.PDUType2.IsFontlist():
		data = pdu.FontListPDUData.Serialize()
	}

	pdu.ShareDataHeader.ShareControlHeader.TotalLength = uint16(18 + len(data)) // #nosec G115
	pdu.ShareDataHeader.UncompressedLength = uint16(4 + len(data))             // #nosec G115

	buf := new(bytes.Buffer)

	buf.Write(pdu.ShareDataHeader.Serialize())
	buf.Write(data)

	return buf.Bytes()
}

// Deserialize decodes the PDU from wire format.
func (pdu *Data) Deserialize(wire io.Reader) error {
	var err error

	if err = pdu.ShareDataHeader.Deserialize(wire); err != nil {
		return err
	}

	switch {
	case pdu.ShareDataHeader.PDUType2.IsSynchronize():
		pdu.SynchronizePDUData = &SynchronizePDUData{}

		return pdu.SynchronizePDUData.Deserialize(wire)
	case pdu.ShareDataHeader.PDUType2.IsControl():
		pdu.ControlPDUData = &ControlPDUData{}

		return pdu.ControlPDUData.Deserialize(wire)
	case pdu.ShareDataHeader.PDUType2.IsFontmap():
		pdu.FontMapPDUData = &FontMapPDUData{}

		return pdu.FontMapPDUData.Deserialize(wire)
	case pdu.ShareDataHeader.PDUType2.IsErrorInfo():
		pdu.ErrorInfoPDUData = &ErrorInfoPDUData{}

		return pdu.ErrorInfoPDUData.Deserialize(wire)
	case pdu.ShareDataHeader.PDUType2.IsSaveSessionInfo(): // ignore
		return nil
	case pdu.ShareDataHeader.PDUType2.IsUpdate(): // slow-path graphics update, handled via fastpath
		return nil
	case pdu.ShareDataHeader.PDUType2.IsPointer(): // pointer update, ignore for now
		return nil
	}

	return fmt.Errorf("unknown data pdu: %d", pdu.ShareDataHeader.PDUType2)
}
