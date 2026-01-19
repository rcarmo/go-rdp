package pdu

import (
"bytes"
"encoding/binary"
"io"
)

type OffscreenBitmapCacheCapabilitySet struct {
	OffscreenSupportLevel uint32
	OffscreenCacheSize    uint16
	OffscreenCacheEntries uint16
}

func NewOffscreenBitmapCacheCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType:                 CapabilitySetTypeOffscreenBitmapCache,
		OffscreenBitmapCacheCapabilitySet: &OffscreenBitmapCacheCapabilitySet{},
	}
}

func (s *OffscreenBitmapCacheCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.OffscreenSupportLevel)
	_ = binary.Write(buf, binary.LittleEndian, s.OffscreenCacheSize)
	_ = binary.Write(buf, binary.LittleEndian, s.OffscreenCacheEntries)

	return buf.Bytes()
}

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
