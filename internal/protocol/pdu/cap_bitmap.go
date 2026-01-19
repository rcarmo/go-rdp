package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
)

type BitmapCapabilitySet struct {
	PreferredBitsPerPixel uint16
	Receive1BitPerPixel   uint16
	Receive4BitsPerPixel  uint16
	Receive8BitsPerPixel  uint16
	DesktopWidth          uint16
	DesktopHeight         uint16
	DesktopResizeFlag     uint16
	DrawingFlags          uint8
}

func NewBitmapCapabilitySet(desktopWidth, desktopHeight uint16) CapabilitySet {
	return CapabilitySet{
		CapabilitySetType: CapabilitySetTypeBitmap,
		BitmapCapabilitySet: &BitmapCapabilitySet{
			PreferredBitsPerPixel: 0x0020, // HIGH_COLOR_32BPP (32-bit color support)
			Receive1BitPerPixel:   0x0001,
			Receive4BitsPerPixel:  0x0001,
			Receive8BitsPerPixel:  0x0001,
			DesktopWidth:          desktopWidth,
			DesktopHeight:         desktopHeight,
			DesktopResizeFlag:     0x0001, // Support dynamic resizing
		},
	}
}

func (s *BitmapCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.PreferredBitsPerPixel)
	_ = binary.Write(buf, binary.LittleEndian, s.Receive1BitPerPixel)
	_ = binary.Write(buf, binary.LittleEndian, s.Receive4BitsPerPixel)
	_ = binary.Write(buf, binary.LittleEndian, s.Receive8BitsPerPixel)
	_ = binary.Write(buf, binary.LittleEndian, s.DesktopWidth)
	_ = binary.Write(buf, binary.LittleEndian, s.DesktopHeight)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // padding
	_ = binary.Write(buf, binary.LittleEndian, s.DesktopResizeFlag)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0001)) // bitmapCompressionFlag
	_ = binary.Write(buf, binary.LittleEndian, uint8(0))       // highColorFlags
	_ = binary.Write(buf, binary.LittleEndian, s.DrawingFlags)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0001)) // multipleRectangleSupport
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))      // padding

	return buf.Bytes()
}

func (s *BitmapCapabilitySet) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &s.PreferredBitsPerPixel)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.Receive1BitPerPixel)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.Receive4BitsPerPixel)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.Receive8BitsPerPixel)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.DesktopWidth)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.DesktopHeight)
	if err != nil {
		return err
	}

	var padding uint16
	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.DesktopResizeFlag)
	if err != nil {
		return err
	}

	var bitmapCompressionFlag uint16
	err = binary.Read(wire, binary.LittleEndian, &bitmapCompressionFlag)
	if err != nil {
		return err
	}

	var highColorFlags uint8
	err = binary.Read(wire, binary.LittleEndian, &highColorFlags)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.DrawingFlags)
	if err != nil {
		return err
	}

	var multipleRectangleSupport uint16
	err = binary.Read(wire, binary.LittleEndian, &multipleRectangleSupport)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	return nil
}
