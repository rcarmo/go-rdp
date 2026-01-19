package pdu

import (
"bytes"
"encoding/binary"
"io"
)

type BitmapCacheHostSupportCapabilitySet struct{}

func NewBitmapCacheHostSupportCapabilitySet() *CapabilitySet {
	return &CapabilitySet{
		CapabilitySetType:                   CapabilitySetTypeBitmapCacheHostSupport,
		BitmapCacheHostSupportCapabilitySet: &BitmapCacheHostSupportCapabilitySet{},
	}
}

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

type ControlCapabilitySet struct{}

func (s *ControlCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint16(0)) // controlFlags
	binary.Write(buf, binary.LittleEndian, uint16(0)) // remoteDetachFlag
	binary.Write(buf, binary.LittleEndian, uint16(2)) // controlInterest
	binary.Write(buf, binary.LittleEndian, uint16(2)) // detachInterest

	return buf.Bytes()
}

func (s *ControlCapabilitySet) Deserialize(wire io.Reader) error {
	padding := make([]byte, 8)

	return binary.Read(wire, binary.LittleEndian, &padding)
}

type WindowActivationCapabilitySet struct{}

func (s *WindowActivationCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint16(0)) // helpKeyFlag
	binary.Write(buf, binary.LittleEndian, uint16(0)) // helpKeyIndexFlag
	binary.Write(buf, binary.LittleEndian, uint16(0)) // helpExtendedKeyFlag
	binary.Write(buf, binary.LittleEndian, uint16(0)) // windowManagerKeyFlag

	return buf.Bytes()
}

func (s *WindowActivationCapabilitySet) Deserialize(wire io.Reader) error {
	padding := make([]byte, 8)

	return binary.Read(wire, binary.LittleEndian, &padding)
}

type ShareCapabilitySet struct{}

func (s *ShareCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, uint16(0)) // nodeID
	binary.Write(buf, binary.LittleEndian, uint16(0)) // pad2octets

	return buf.Bytes()
}

func (s *ShareCapabilitySet) Deserialize(wire io.Reader) error {
	padding := make([]byte, 4)

	return binary.Read(wire, binary.LittleEndian, &padding)
}

type FontCapabilitySet struct {
	fontSupportFlags uint16
}

func (s *FontCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, s.fontSupportFlags)
	binary.Write(buf, binary.LittleEndian, uint16(0)) // padding

	return buf.Bytes()
}

func (s *FontCapabilitySet) Deserialize(wire io.Reader) error {
	padding := make([]byte, 2)

	err := binary.Read(wire, binary.LittleEndian, &s.fontSupportFlags)
	if err != nil {
		return err
	}

	return binary.Read(wire, binary.LittleEndian, &padding)
}
