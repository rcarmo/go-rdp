package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
)

// GeneralCapabilitySet represents the General Capability Set (MS-RDPBCGR 2.2.7.1.1).
type GeneralCapabilitySet struct {
	OSMajorType           uint16
	OSMinorType           uint16
	ExtraFlags            uint16
	RefreshRectSupport    uint8
	SuppressOutputSupport uint8
}

// NewGeneralCapabilitySet creates a General Capability Set with default client values.
func NewGeneralCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType: CapabilitySetTypeGeneral,
		GeneralCapabilitySet: &GeneralCapabilitySet{
			OSMajorType:           0x000A,                                     // Windows 10+ platform
			OSMinorType:           0x0000,                                     // Latest version
			ExtraFlags:            0x0001 | 0x0004 | 0x0400 | 0x0080 | 0x0100, // Enhanced features: FASTPATH_OUTPUT_SUPPORTED, LONG_CREDENTIALS_SUPPORTED, NO_BITMAP_COMPRESSION_HDR, DYNAMIC_DST_SUPPORTED, TILE_SUPPORT
			RefreshRectSupport:    1,                                          // We support Refresh Rect PDU
			SuppressOutputSupport: 1,                                          // We support Suppress Output PDU
		},
	}
}

// Serialize encodes the capability set to wire format.
func (s *GeneralCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.OSMajorType)
	_ = binary.Write(buf, binary.LittleEndian, s.OSMinorType)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0200)) // protocolVersion
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // padding
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // compressionTypes
	_ = binary.Write(buf, binary.LittleEndian, s.ExtraFlags)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // updateCapabilityFlag
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // remoteUnshareFlag
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000)) // compressionLevel
	_ = binary.Write(buf, binary.LittleEndian, s.RefreshRectSupport)
	_ = binary.Write(buf, binary.LittleEndian, s.SuppressOutputSupport)

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
func (s *GeneralCapabilitySet) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &s.OSMajorType)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.OSMinorType)
	if err != nil {
		return err
	}

	var protocolVersion uint16
	err = binary.Read(wire, binary.LittleEndian, &protocolVersion)
	if err != nil {
		return err
	}

	var padding uint16
	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	var compressionTypes uint16
	err = binary.Read(wire, binary.LittleEndian, &compressionTypes)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.ExtraFlags)
	if err != nil {
		return err
	}

	var updateCapabilityFlag uint16
	err = binary.Read(wire, binary.LittleEndian, &updateCapabilityFlag)
	if err != nil {
		return err
	}

	var remoteUnshareFlag uint16
	err = binary.Read(wire, binary.LittleEndian, &remoteUnshareFlag)
	if err != nil {
		return err
	}

	var compressionLevel uint16
	err = binary.Read(wire, binary.LittleEndian, &compressionLevel)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.RefreshRectSupport)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.SuppressOutputSupport)
	if err != nil {
		return err
	}

	return nil
}
