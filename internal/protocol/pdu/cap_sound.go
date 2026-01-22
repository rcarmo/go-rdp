package pdu

import (
"bytes"
"encoding/binary"
"io"
)

// SoundCapabilitySet represents the Sound Capability Set (MS-RDPBCGR 2.2.7.1.11).
type SoundCapabilitySet struct {
	SoundFlags uint16
}

// NewSoundCapabilitySet creates a Sound Capability Set with default values.
func NewSoundCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType:  CapabilitySetTypeSound,
		SoundCapabilitySet: &SoundCapabilitySet{},
	}
}

// Serialize encodes the capability set to wire format.
func (s *SoundCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.SoundFlags)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
func (s *SoundCapabilitySet) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &s.SoundFlags)
	if err != nil {
		return err
	}

	var padding uint16
	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	return nil
}
