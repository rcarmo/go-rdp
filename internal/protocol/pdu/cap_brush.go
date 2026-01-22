package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
)

// BrushSupportLevel indicates the level of brush support as defined in MS-RDPBCGR section 2.2.7.1.7.
type BrushSupportLevel uint32

const (
	// BrushSupportLevelDefault BRUSH_DEFAULT
	BrushSupportLevelDefault BrushSupportLevel = 0

	// BrushSupportLevelColor8x8 BRUSH_COLOR_8x8
	BrushSupportLevelColor8x8 BrushSupportLevel = 1

	// BrushSupportLevelFull BRUSH_COLOR_FULL
	BrushSupportLevelFull BrushSupportLevel = 2
)

// BrushCapabilitySet advertises brush capabilities as defined in MS-RDPBCGR section 2.2.7.1.7.
type BrushCapabilitySet struct {
	BrushSupportLevel BrushSupportLevel
}

// NewBrushCapabilitySet creates a BrushCapabilitySet with default values.
func NewBrushCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType:  CapabilitySetTypeBrush,
		BrushCapabilitySet: &BrushCapabilitySet{},
	}
}

// Serialize encodes the BrushCapabilitySet to wire format.
func (s *BrushCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint32(s.BrushSupportLevel))

	return buf.Bytes()
}

// Deserialize decodes the BrushCapabilitySet from wire format.
func (s *BrushCapabilitySet) Deserialize(wire io.Reader) error {
	return binary.Read(wire, binary.LittleEndian, &s.BrushSupportLevel)
}

// CacheDefinition describes a glyph cache entry as defined in MS-RDPBCGR section 2.2.7.1.8.
type CacheDefinition struct {
	CacheEntries         uint16
	CacheMaximumCellSize uint16
}

// Serialize encodes the CacheDefinition to wire format.
func (d *CacheDefinition) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, d.CacheEntries)
	_ = binary.Write(buf, binary.LittleEndian, d.CacheMaximumCellSize)

	return buf.Bytes()
}

// Deserialize decodes the CacheDefinition from wire format.
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
