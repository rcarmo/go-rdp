package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
)

type BitmapCacheCapabilitySetRev1 struct {
	Cache0Entries         uint16
	Cache0MaximumCellSize uint16
	Cache1Entries         uint16
	Cache1MaximumCellSize uint16
	Cache2Entries         uint16
	Cache2MaximumCellSize uint16
}

func NewBitmapCacheCapabilitySetRev1() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType:            CapabilitySetTypeBitmapCache,
		BitmapCacheCapabilitySetRev1: &BitmapCacheCapabilitySetRev1{},
	}
}

func (s *BitmapCacheCapabilitySetRev1) Serialize() []byte {
	buf := new(bytes.Buffer)

	buf.Write(make([]byte, 24)) // padding
	_ = binary.Write(buf, binary.LittleEndian, &s.Cache0Entries)
	_ = binary.Write(buf, binary.LittleEndian, &s.Cache0MaximumCellSize)
	_ = binary.Write(buf, binary.LittleEndian, &s.Cache1Entries)
	_ = binary.Write(buf, binary.LittleEndian, &s.Cache1MaximumCellSize)
	_ = binary.Write(buf, binary.LittleEndian, &s.Cache2Entries)
	_ = binary.Write(buf, binary.LittleEndian, &s.Cache2MaximumCellSize)

	return buf.Bytes()
}

func (s *BitmapCacheCapabilitySetRev1) Deserialize(wire io.Reader) error {
	var (
		padding [24]byte
		err     error
	)

	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.Cache0Entries)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.Cache0MaximumCellSize)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.Cache1Entries)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.Cache1MaximumCellSize)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.Cache2Entries)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.Cache2MaximumCellSize)
	if err != nil {
		return err
	}

	return nil
}

type BitmapCacheCapabilitySetRev2 struct {
	CacheFlags           uint16
	NumCellCaches        uint8
	BitmapCache0CellInfo uint32
	BitmapCache1CellInfo uint32
	BitmapCache2CellInfo uint32
	BitmapCache3CellInfo uint32
	BitmapCache4CellInfo uint32
}

func NewBitmapCacheCapabilitySetRev2() *CapabilitySet {
	return &CapabilitySet{
		CapabilitySetType:            CapabilitySetTypeBitmapCacheRev2,
		BitmapCacheCapabilitySetRev2: &BitmapCacheCapabilitySetRev2{},
	}
}

func (s *BitmapCacheCapabilitySetRev2) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, &s.CacheFlags)
	_ = binary.Write(buf, binary.LittleEndian, uint8(0)) // padding
	_ = binary.Write(buf, binary.LittleEndian, &s.NumCellCaches)
	_ = binary.Write(buf, binary.LittleEndian, &s.BitmapCache0CellInfo)
	_ = binary.Write(buf, binary.LittleEndian, &s.BitmapCache1CellInfo)
	_ = binary.Write(buf, binary.LittleEndian, &s.BitmapCache2CellInfo)
	_ = binary.Write(buf, binary.LittleEndian, &s.BitmapCache3CellInfo)
	_ = binary.Write(buf, binary.LittleEndian, &s.BitmapCache4CellInfo)
	buf.Write(make([]byte, 12)) // padding

	return buf.Bytes()
}

func (s *BitmapCacheCapabilitySetRev2) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &s.CacheFlags)
	if err != nil {
		return err
	}

	var padding uint8
	err = binary.Read(wire, binary.LittleEndian, &padding)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.NumCellCaches)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.BitmapCache0CellInfo)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.BitmapCache1CellInfo)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.BitmapCache2CellInfo)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.BitmapCache3CellInfo)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.BitmapCache4CellInfo)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &s.BitmapCache4CellInfo)
	if err != nil {
		return err
	}

	var padding2 [12]byte
	err = binary.Read(wire, binary.LittleEndian, &padding2)
	if err != nil {
		return err
	}

	return nil
}

type ColorCacheCapabilitySet struct {
	ColorTableCacheSize uint16
}

func (s *ColorCacheCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, &s.ColorTableCacheSize)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // padding

	return buf.Bytes()
}

func (s *ColorCacheCapabilitySet) Deserialize(wire io.Reader) error {
	var (
		padding uint16
		err     error
	)

	err = binary.Read(wire, binary.LittleEndian, &s.ColorTableCacheSize)
	if err != nil {
		return err
	}

	return binary.Read(wire, binary.LittleEndian, &padding)
}
