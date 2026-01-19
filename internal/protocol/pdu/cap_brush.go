package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
)

type BrushSupportLevel uint32

const (
	// BrushSupportLevelDefault BRUSH_DEFAULT
	BrushSupportLevelDefault BrushSupportLevel = 0

	// BrushSupportLevelColor8x8 BRUSH_COLOR_8x8
	BrushSupportLevelColor8x8 BrushSupportLevel = 1

	// BrushSupportLevelFull BRUSH_COLOR_FULL
	BrushSupportLevelFull BrushSupportLevel = 2
)

type BrushCapabilitySet struct {
	BrushSupportLevel BrushSupportLevel
}

func NewBrushCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType:  CapabilitySetTypeBrush,
		BrushCapabilitySet: &BrushCapabilitySet{},
	}
}

func (s *BrushCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint32(s.BrushSupportLevel))

	return buf.Bytes()
}

func (s *BrushCapabilitySet) Deserialize(wire io.Reader) error {
	return binary.Read(wire, binary.LittleEndian, &s.BrushSupportLevel)
}

type CacheDefinition struct {
	CacheEntries         uint16
	CacheMaximumCellSize uint16
}

func (d *CacheDefinition) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, d.CacheEntries)
	_ = binary.Write(buf, binary.LittleEndian, d.CacheMaximumCellSize)

	return buf.Bytes()
}

func (d *CacheDefinition) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &d.CacheEntries)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &d.CacheMaximumCellSize)
	if err != nil {
		return err
	}

	return nil
}
