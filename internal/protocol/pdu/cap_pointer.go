package pdu

import (
"bytes"
"encoding/binary"
"io"
)

type PointerCapabilitySet struct {
	ColorPointerFlag      uint16
	ColorPointerCacheSize uint16
	PointerCacheSize      uint16
	lengthCapability      uint16
}

func NewPointerCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType: CapabilitySetTypePointer,
		PointerCapabilitySet: &PointerCapabilitySet{
			ColorPointerFlag: 1, // advertise color cursor support
			PointerCacheSize: 25,
		},
	}
}

func (s *PointerCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.ColorPointerFlag)
	_ = binary.Write(buf, binary.LittleEndian, s.ColorPointerCacheSize)
	_ = binary.Write(buf, binary.LittleEndian, s.PointerCacheSize)

	return buf.Bytes()
}

func (s *PointerCapabilitySet) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &s.ColorPointerFlag)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.ColorPointerCacheSize)
	if err != nil {
		return err
	}

	if s.lengthCapability == 4 {
		return nil
	}

	err = binary.Read(wire, binary.LittleEndian, &s.PointerCacheSize)
	if err != nil {
		return err
	}

	return nil
}
