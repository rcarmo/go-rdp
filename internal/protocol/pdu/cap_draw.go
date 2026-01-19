package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
)

type DrawNineGridCacheCapabilitySet struct {
	drawNineGridSupportLevel uint32
	drawNineGridCacheSize    uint16
	drawNineGridCacheEntries uint16
}

func (s *DrawNineGridCacheCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, s.drawNineGridSupportLevel)
	binary.Write(buf, binary.LittleEndian, s.drawNineGridCacheSize)
	binary.Write(buf, binary.LittleEndian, s.drawNineGridCacheEntries)

	return buf.Bytes()
}

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

type GDICacheEntries struct {
	GdipGraphicsCacheEntries        uint16
	GdipBrushCacheEntries           uint16
	GdipPenCacheEntries             uint16
	GdipImageCacheEntries           uint16
	GdipImageAttributesCacheEntries uint16
}

func (e *GDICacheEntries) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, e.GdipGraphicsCacheEntries)
	binary.Write(buf, binary.LittleEndian, e.GdipBrushCacheEntries)
	binary.Write(buf, binary.LittleEndian, e.GdipPenCacheEntries)
	binary.Write(buf, binary.LittleEndian, e.GdipImageCacheEntries)
	binary.Write(buf, binary.LittleEndian, e.GdipImageAttributesCacheEntries)

	return buf.Bytes()
}

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

type GDICacheChunkSize struct {
	GdipGraphicsCacheChunkSize              uint16
	GdipObjectBrushCacheChunkSize           uint16
	GdipObjectPenCacheChunkSize             uint16
	GdipObjectImageAttributesCacheChunkSize uint16
}

func (s *GDICacheChunkSize) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, s.GdipGraphicsCacheChunkSize)
	binary.Write(buf, binary.LittleEndian, s.GdipObjectBrushCacheChunkSize)
	binary.Write(buf, binary.LittleEndian, s.GdipObjectPenCacheChunkSize)
	binary.Write(buf, binary.LittleEndian, s.GdipObjectImageAttributesCacheChunkSize)

	return buf.Bytes()
}

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

type GDIImageCacheProperties struct {
	GdipObjectImageCacheChunkSize uint16
	GdipObjectImageCacheTotalSize uint16
	GdipObjectImageCacheMaxSize   uint16
}

func (p *GDIImageCacheProperties) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, p.GdipObjectImageCacheChunkSize)
	binary.Write(buf, binary.LittleEndian, p.GdipObjectImageCacheTotalSize)
	binary.Write(buf, binary.LittleEndian, p.GdipObjectImageCacheMaxSize)

	return buf.Bytes()
}

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

type DrawGDIPlusCapabilitySet struct {
	drawGDIPlusSupportLevel  uint32
	GdipVersion              uint32
	drawGdiplusCacheLevel    uint32
	GdipCacheEntries         GDICacheEntries
	GdipCacheChunkSize       GDICacheChunkSize
	GdipImageCacheProperties GDIImageCacheProperties
}

func (s *DrawGDIPlusCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, s.drawGDIPlusSupportLevel)
	binary.Write(buf, binary.LittleEndian, s.GdipVersion)
	binary.Write(buf, binary.LittleEndian, s.drawGdiplusCacheLevel)

	buf.Write(s.GdipCacheEntries.Serialize())
	buf.Write(s.GdipCacheChunkSize.Serialize())
	buf.Write(s.GdipImageCacheProperties.Serialize())

	return buf.Bytes()
}

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
