package pdu

import (
"bytes"
"encoding/binary"
"io"
)

// OffscreenBitmapCacheCapabilitySet represents the TS_OFFSCREEN_CAPABILITYSET
// structure (MS-RDPBCGR 2.2.7.2.6).
type OffscreenBitmapCacheCapabilitySet struct {
	OffscreenSupportLevel uint32
	OffscreenCacheSize    uint16
	OffscreenCacheEntries uint16
}

// NewOffscreenBitmapCacheCapabilitySet creates a new OffscreenBitmapCacheCapabilitySet.
func NewOffscreenBitmapCacheCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType:                 CapabilitySetTypeOffscreenBitmapCache,
		OffscreenBitmapCacheCapabilitySet: &OffscreenBitmapCacheCapabilitySet{},
	}
}

// Serialize encodes the capability set to wire format.
func (s *OffscreenBitmapCacheCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.OffscreenSupportLevel)
	_ = binary.Write(buf, binary.LittleEndian, s.OffscreenCacheSize)
	_ = binary.Write(buf, binary.LittleEndian, s.OffscreenCacheEntries)

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
func (s *OffscreenBitmapCacheCapabilitySet) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &s.OffscreenSupportLevel)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.OffscreenCacheSize)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.OffscreenCacheEntries)
	if err != nil {
		return err
	}

	return nil
}
