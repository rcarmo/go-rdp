package pdu

import (
"bytes"
"encoding/binary"
"io"
)

type SoundCapabilitySet struct {
	SoundFlags uint16
}

func NewSoundCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType:  CapabilitySetTypeSound,
		SoundCapabilitySet: &SoundCapabilitySet{},
	}
}

func (s *SoundCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.SoundFlags)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))

	return buf.Bytes()
}

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
