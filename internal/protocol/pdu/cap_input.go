package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
)

// InputCapabilitySet represents the Input Capability Set (MS-RDPBCGR 2.2.7.1.6).
type InputCapabilitySet struct {
	InputFlags          uint16
	KeyboardLayout      uint32
	KeyboardType        uint32
	KeyboardSubType     uint32
	KeyboardFunctionKey uint32
	ImeFileName         [64]byte
}

// NewInputCapabilitySet creates an Input Capability Set with default client values.
func NewInputCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType: CapabilitySetTypeInput,
		InputCapabilitySet: &InputCapabilitySet{
			InputFlags:          0x0001 | 0x0004 | 0x0010 | 0x0020, // INPUT_FLAG_SCANCODES, INPUT_FLAG_MOUSEX, INPUT_FLAG_UNICODE, INPUT_FLAG_FASTPATH_INPUT2
			KeyboardLayout:      0x00000409,                        // US
			KeyboardType:        0x00000004,                        // IBM enhanced (101- or 102-key) keyboard
			KeyboardFunctionKey: 12,
		},
	}
}

// Serialize encodes the capability set to wire format.
func (s *InputCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.InputFlags)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // padding
	_ = binary.Write(buf, binary.LittleEndian, s.KeyboardLayout)
	_ = binary.Write(buf, binary.LittleEndian, s.KeyboardType)
	_ = binary.Write(buf, binary.LittleEndian, s.KeyboardSubType)
	_ = binary.Write(buf, binary.LittleEndian, s.KeyboardFunctionKey)
	_ = binary.Write(buf, binary.LittleEndian, s.ImeFileName)

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
func (s *InputCapabilitySet) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &s.InputFlags)
	if err != nil {
		return err
	}

	var padding uint16
	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.KeyboardLayout)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.KeyboardType)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.KeyboardSubType)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.KeyboardFunctionKey)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.ImeFileName)
	if err != nil {
		return err
	}

	return nil
}
