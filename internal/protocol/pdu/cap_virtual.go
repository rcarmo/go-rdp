package pdu

import (
"bytes"
"encoding/binary"
"io"
)

// VirtualChannelCapabilitySet represents the TS_VIRTUALCHANNEL_CAPABILITYSET
// structure (MS-RDPBCGR 2.2.7.1.10).
type VirtualChannelCapabilitySet struct {
	Flags       uint32
	VCChunkSize uint32
}

// NewVirtualChannelCapabilitySet creates a new VirtualChannelCapabilitySet.
func NewVirtualChannelCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType:           CapabilitySetTypeVirtualChannel,
		VirtualChannelCapabilitySet: &VirtualChannelCapabilitySet{},
	}
}

// Serialize encodes the capability set to wire format.
func (s *VirtualChannelCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.Flags)
	_ = binary.Write(buf, binary.LittleEndian, s.VCChunkSize)

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
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
