package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
)

type GlyphSupportLevel uint16

const (
	// GlyphSupportLevelNone GLYPH_SUPPORT_NONE
	GlyphSupportLevelNone GlyphSupportLevel = 0

	// GlyphSupportLevelPartial GLYPH_SUPPORT_PARTIAL
	GlyphSupportLevelPartial GlyphSupportLevel = 1

	// GlyphSupportLevelFull GLYPH_SUPPORT_FULL
	GlyphSupportLevelFull GlyphSupportLevel = 2

	// GlyphSupportLevelEncode GLYPH_SUPPORT_ENCODE
	GlyphSupportLevelEncode GlyphSupportLevel = 3
)

type GlyphCacheCapabilitySet struct {
	GlyphCache        [10]CacheDefinition
	FragCache         uint32
	GlyphSupportLevel GlyphSupportLevel
}

func NewGlyphCacheCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType:       CapabilitySetTypeGlyphCache,
		GlyphCacheCapabilitySet: &GlyphCacheCapabilitySet{},
	}
}

func (s *GlyphCacheCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	for i := range s.GlyphCache {
		buf.Write(s.GlyphCache[i].Serialize())
	}

	_ = binary.Write(buf, binary.LittleEndian, s.FragCache)
	_ = binary.Write(buf, binary.LittleEndian, s.GlyphSupportLevel)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // padding

	return buf.Bytes()
}

func (s *GlyphCacheCapabilitySet) Deserialize(wire io.Reader) error {
	var err error

	for i := range s.GlyphCache {
		err = s.GlyphCache[i].Deserialize(wire)
		if err != nil {
			return err
		}
	}

	err = binary.Read(wire, binary.LittleEndian, &s.FragCache)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.GlyphSupportLevel)
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
