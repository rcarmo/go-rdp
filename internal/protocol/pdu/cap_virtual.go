package pdu

import (
"bytes"
"encoding/binary"
"io"
)

type VirtualChannelCapabilitySet struct {
	Flags       uint32
	VCChunkSize uint32
}

func NewVirtualChannelCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType:           CapabilitySetTypeVirtualChannel,
		VirtualChannelCapabilitySet: &VirtualChannelCapabilitySet{},
	}
}

func (s *VirtualChannelCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.Flags)
	_ = binary.Write(buf, binary.LittleEndian, s.VCChunkSize)

	return buf.Bytes()
}

func (s *VirtualChannelCapabilitySet) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &s.Flags)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.VCChunkSize)
	if err != nil {
		return err
	}

	return nil
}
