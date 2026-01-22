package pdu

import (
"bytes"
"encoding/binary"
"io"
)

// OrderCapabilitySet represents the Order Capability Set (MS-RDPBCGR 2.2.7.1.3).
type OrderCapabilitySet struct {
	OrderFlags          uint16
	OrderSupport        [32]byte
	textFlags           uint16
	OrderSupportExFlags uint16
	DesktopSaveSize     uint32
	textANSICodePage    uint16
}

// NewOrderCapabilitySet creates an Order Capability Set with default client values.
func NewOrderCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType: CapabilitySetTypeOrder,
		OrderCapabilitySet: &OrderCapabilitySet{
			OrderFlags:      0x2 | 0x0008, // NEGOTIATEORDERSUPPORT, ZEROBOUNDSDELTASSUPPORT this flags must be set
			DesktopSaveSize: 480 * 480,
		},
	}
}

// Serialize encodes the capability set to wire format.
func (s *OrderCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.Write(make([]byte, 16))                            // terminalDescriptor
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))  // padding
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))  // desktopSaveXGranularity
	_ = binary.Write(buf, binary.LittleEndian, uint16(20)) // desktopSaveYGranularity
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))  // padding
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))  // maximumOrderLevel = ORD_LEVEL_1_ORDERS
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))  // numberFonts
	_ = binary.Write(buf, binary.LittleEndian, s.OrderFlags)
	_ = binary.Write(buf, binary.LittleEndian, s.OrderSupport)
	_ = binary.Write(buf, binary.LittleEndian, s.textFlags) // textFlags
	_ = binary.Write(buf, binary.LittleEndian, s.OrderSupportExFlags)
	_ = binary.Write(buf, binary.LittleEndian, uint32(0)) // padding
	_ = binary.Write(buf, binary.LittleEndian, s.DesktopSaveSize)
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))          // padding
	_ = binary.Write(buf, binary.LittleEndian, s.textANSICodePage) // textANSICodePage
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))          // padding

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
func (s *OrderCapabilitySet) Deserialize(wire io.Reader) error {
	var (
		err                     error
		terminalDescriptor      [16]byte
		padding                 uint32
		desktopSaveXGranularity uint16
		desktopSaveYGranularity uint16
		padding2                uint16
		maximumOrderLevel       uint16
		numberFonts             uint16
		textFlags               uint16
		textANSICodePage        uint16
	)

	err = binary.Read(wire, binary.LittleEndian, &terminalDescriptor)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &desktopSaveXGranularity)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &desktopSaveYGranularity)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &padding2)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &maximumOrderLevel)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &numberFonts)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.OrderFlags)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.OrderSupport)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &textFlags)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.OrderSupportExFlags)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.DesktopSaveSize)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &textANSICodePage)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &padding2)
	if err != nil {
		return err
	}

	return nil
}
