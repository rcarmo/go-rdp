package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
)

// DrawNineGridCacheCapabilitySet represents the TS_DRAW_NINEGRID_CAPABILITYSET
// structure (MS-RDPBCGR 2.2.7.2.8).
type DrawNineGridCacheCapabilitySet struct {
	drawNineGridSupportLevel uint32
	drawNineGridCacheSize    uint16
	drawNineGridCacheEntries uint16
}

// Serialize encodes the capability set to wire format.
func (s *DrawNineGridCacheCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.drawNineGridSupportLevel)
	_ = binary.Write(buf, binary.LittleEndian, s.drawNineGridCacheSize)
	_ = binary.Write(buf, binary.LittleEndian, s.drawNineGridCacheEntries)

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
func (s *DrawNineGridCacheCapabilitySet) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &s.drawNineGridSupportLevel)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.drawNineGridCacheSize)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.drawNineGridCacheEntries)
	if err != nil {
		return err
	}

	return nil
}

// GDICacheEntries contains GDI+ cache entry counts for the DrawGDIPlus capability set.
type GDICacheEntries struct {
	GdipGraphicsCacheEntries        uint16
	GdipBrushCacheEntries           uint16
	GdipPenCacheEntries             uint16
	GdipImageCacheEntries           uint16
	GdipImageAttributesCacheEntries uint16
}

// Serialize encodes the cache entries to wire format.
func (e *GDICacheEntries) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, e.GdipGraphicsCacheEntries)
	_ = binary.Write(buf, binary.LittleEndian, e.GdipBrushCacheEntries)
	_ = binary.Write(buf, binary.LittleEndian, e.GdipPenCacheEntries)
	_ = binary.Write(buf, binary.LittleEndian, e.GdipImageCacheEntries)
	_ = binary.Write(buf, binary.LittleEndian, e.GdipImageAttributesCacheEntries)

	return buf.Bytes()
}

// Deserialize decodes the cache entries from wire format.
func (e *GDICacheEntries) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &e.GdipGraphicsCacheEntries)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &e.GdipBrushCacheEntries)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &e.GdipPenCacheEntries)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &e.GdipImageCacheEntries)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &e.GdipImageAttributesCacheEntries)
	if err != nil {
		return err
	}

	return nil
}

// GDICacheChunkSize contains GDI+ cache chunk sizes for the DrawGDIPlus capability set.
type GDICacheChunkSize struct {
	GdipGraphicsCacheChunkSize              uint16
	GdipObjectBrushCacheChunkSize           uint16
	GdipObjectPenCacheChunkSize             uint16
	GdipObjectImageAttributesCacheChunkSize uint16
}

// Serialize encodes the cache chunk sizes to wire format.
func (s *GDICacheChunkSize) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.GdipGraphicsCacheChunkSize)
	_ = binary.Write(buf, binary.LittleEndian, s.GdipObjectBrushCacheChunkSize)
	_ = binary.Write(buf, binary.LittleEndian, s.GdipObjectPenCacheChunkSize)
	_ = binary.Write(buf, binary.LittleEndian, s.GdipObjectImageAttributesCacheChunkSize)

	return buf.Bytes()
}

// Deserialize decodes the cache chunk sizes from wire format.
func (s *GDICacheChunkSize) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &s.GdipGraphicsCacheChunkSize)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.GdipObjectBrushCacheChunkSize)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.GdipObjectPenCacheChunkSize)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.GdipObjectImageAttributesCacheChunkSize)
	if err != nil {
		return err
	}

	return nil
}

// GDIImageCacheProperties contains GDI+ image cache properties for the DrawGDIPlus capability set.
type GDIImageCacheProperties struct {
	GdipObjectImageCacheChunkSize uint16
	GdipObjectImageCacheTotalSize uint16
	GdipObjectImageCacheMaxSize   uint16
}

// Serialize encodes the image cache properties to wire format.
func (p *GDIImageCacheProperties) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, p.GdipObjectImageCacheChunkSize)
	_ = binary.Write(buf, binary.LittleEndian, p.GdipObjectImageCacheTotalSize)
	_ = binary.Write(buf, binary.LittleEndian, p.GdipObjectImageCacheMaxSize)

	return buf.Bytes()
}

// Deserialize decodes the image cache properties from wire format.
func (p *GDIImageCacheProperties) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &p.GdipObjectImageCacheChunkSize)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &p.GdipObjectImageCacheTotalSize)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &p.GdipObjectImageCacheMaxSize)
	if err != nil {
		return err
	}

	return nil
}

// DrawGDIPlusCapabilitySet represents the TS_DRAW_GDIPLUS_CAPABILITYSET
// structure (MS-RDPBCGR 2.2.7.2.9).
type DrawGDIPlusCapabilitySet struct {
	drawGDIPlusSupportLevel  uint32
	GdipVersion              uint32
	drawGdiplusCacheLevel    uint32
	GdipCacheEntries         GDICacheEntries
	GdipCacheChunkSize       GDICacheChunkSize
	GdipImageCacheProperties GDIImageCacheProperties
}

// Serialize encodes the capability set to wire format.
func (s *DrawGDIPlusCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.drawGDIPlusSupportLevel)
	_ = binary.Write(buf, binary.LittleEndian, s.GdipVersion)
	_ = binary.Write(buf, binary.LittleEndian, s.drawGdiplusCacheLevel)

	buf.Write(s.GdipCacheEntries.Serialize())
	buf.Write(s.GdipCacheChunkSize.Serialize())
	buf.Write(s.GdipImageCacheProperties.Serialize())

	return buf.Bytes()
}

// Deserialize decodes the capability set from wire format.
func (s *DrawGDIPlusCapabilitySet) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &s.drawGDIPlusSupportLevel)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.GdipVersion)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.drawGdiplusCacheLevel)
	if err != nil {
		return err
	}

	err = s.GdipCacheEntries.Deserialize(wire)
	if err != nil {
		return err
	}

	err = s.GdipCacheChunkSize.Deserialize(wire)
	if err != nil {
		return err
	}

	err = s.GdipImageCacheProperties.Deserialize(wire)
	if err != nil {
		return err
	}

	return nil
}
