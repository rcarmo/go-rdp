package pdu

import (
"bytes"
"encoding/binary"
"io"
)

// BitmapCacheHostSupportCapabilitySet represents the TS_BITMAPCACHE_HOSTSUPPORT_CAPABILITYSET
// structure (MS-RDPBCGR 2.2.7.2.1).
type BitmapCacheHostSupportCapabilitySet struct{}

// NewBitmapCacheHostSupportCapabilitySet creates a new BitmapCacheHostSupportCapabilitySet.
func NewBitmapCacheHostSupportCapabilitySet() *CapabilitySet {
	return &CapabilitySet{
		CapabilitySetType:                   CapabilitySetTypeBitmapCacheHostSupport,
		BitmapCacheHostSupportCapabilitySet: &BitmapCacheHostSupportCapabilitySet{},
	}
}

// Deserialize decodes the capability set from wire format.
func (s *BitmapCacheHostSupportCapabilitySet) Deserialize(wire io.Reader) error {
	var (
		cacheVersion uint8
		padding1     uint8
		padding2     uint16
		err          error
	)

	err = binary.Read(wire, binary.LittleEndian, &cacheVersion)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &padding1)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &padding2)
	if err != nil {
		return err
	}

	return err
}

// ControlCapabilitySet represents the TS_CONTROL_CAPABILITYSET structure (MS-RDPBCGR 2.2.7.2.2).
type ControlCapabilitySet struct{}

// Serialize encodes the capability set to wire format.
func (s *ControlCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // controlFlags
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // remoteDetachFlag
	_ = binary.Write(buf, binary.LittleEndian, uint16(2)) // controlInterest
	_ = binary.Write(buf, binary.LittleEndian, uint16(2)) // detachInterest

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
func (s *ControlCapabilitySet) Deserialize(wire io.Reader) error {
	padding := make([]byte, 8)

	return binary.Read(wire, binary.LittleEndian, &padding)
}

// WindowActivationCapabilitySet represents the TS_WINDOWACTIVATION_CAPABILITYSET
// structure (MS-RDPBCGR 2.2.7.2.3).
type WindowActivationCapabilitySet struct{}

// Serialize encodes the capability set to wire format.
func (s *WindowActivationCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // helpKeyFlag
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // helpKeyIndexFlag
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // helpExtendedKeyFlag
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // windowManagerKeyFlag

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
func (s *WindowActivationCapabilitySet) Deserialize(wire io.Reader) error {
	padding := make([]byte, 8)

	return binary.Read(wire, binary.LittleEndian, &padding)
}

// ShareCapabilitySet represents the TS_SHARE_CAPABILITYSET structure (MS-RDPBCGR 2.2.7.2.4).
type ShareCapabilitySet struct{}

// Serialize encodes the capability set to wire format.
func (s *ShareCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // nodeID
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // pad2octets

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
func (s *ShareCapabilitySet) Deserialize(wire io.Reader) error {
	padding := make([]byte, 4)

	return binary.Read(wire, binary.LittleEndian, &padding)
}

// FontCapabilitySet represents the TS_FONT_CAPABILITYSET structure (MS-RDPBCGR 2.2.7.2.5).
type FontCapabilitySet struct {
	fontSupportFlags uint16
}

// Serialize encodes the capability set to wire format.
func (s *FontCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.fontSupportFlags)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // padding

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
func (s *FontCapabilitySet) Deserialize(wire io.Reader) error {
	padding := make([]byte, 2)

	err := binary.Read(wire, binary.LittleEndian, &s.fontSupportFlags)
	if err != nil {
		return err
	}

	return binary.Read(wire, binary.LittleEndian, &padding)
}
