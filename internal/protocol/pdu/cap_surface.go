package pdu

import (
"bytes"
"encoding/binary"
"io"
)

type MultifragmentUpdateCapabilitySet struct {
	MaxRequestSize uint32
}

func NewMultifragmentUpdateCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType:                CapabilitySetTypeMultifragmentUpdate,
		MultifragmentUpdateCapabilitySet: &MultifragmentUpdateCapabilitySet{},
	}
}

func (s *MultifragmentUpdateCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, &s.MaxRequestSize)

	return buf.Bytes()
}

func (s *MultifragmentUpdateCapabilitySet) Deserialize(wire io.Reader) error {
	return binary.Read(wire, binary.LittleEndian, &s.MaxRequestSize)
}

type LargePointerCapabilitySet struct {
	LargePointerSupportFlags uint16
}

func (s *LargePointerCapabilitySet) Deserialize(wire io.Reader) error {
	return binary.Read(wire, binary.LittleEndian, &s.LargePointerSupportFlags)
}

type DesktopCompositionCapabilitySet struct {
	CompDeskSupportLevel uint16
}

func (s *DesktopCompositionCapabilitySet) Deserialize(wire io.Reader) error {
	return binary.Read(wire, binary.LittleEndian, &s.CompDeskSupportLevel)
}

type SurfaceCommandsCapabilitySet struct {
	CmdFlags uint32
}

// Surface command flags
const (
	SurfCmdSetSurfaceBits  uint32 = 0x00000002
	SurfCmdFrameMarker     uint32 = 0x00000010
	SurfCmdStreamSurfBits  uint32 = 0x00000040
)

func NewSurfaceCommandsCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType: CapabilitySetTypeSurfaceCommands,
		SurfaceCommandsCapabilitySet: &SurfaceCommandsCapabilitySet{
			CmdFlags: SurfCmdSetSurfaceBits | SurfCmdFrameMarker | SurfCmdStreamSurfBits,
		},
	}
}

func (s *SurfaceCommandsCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, s.CmdFlags)
	_ = binary.Write(buf, binary.LittleEndian, uint32(0)) // reserved
	return buf.Bytes()
}

func (s *SurfaceCommandsCapabilitySet) Deserialize(wire io.Reader) error {
	var (
		reserved uint32
		err      error
	)

	err = binary.Read(wire, binary.LittleEndian, &s.CmdFlags)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &reserved)
	if err != nil {
		return err
	}

	return nil
}

type BitmapCodec struct {
	CodecGUID       [16]byte
	CodecID         uint8
	CodecProperties []byte
}

func (c *BitmapCodec) Deserialize(wire io.Reader) error {
	var err error

	err = binary.Read(wire, binary.LittleEndian, &c.CodecGUID)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &c.CodecID)
	if err != nil {
		return err
	}

	var codecPropertiesLength uint16

	err = binary.Read(wire, binary.LittleEndian, &codecPropertiesLength)
	if err != nil {
		return err
	}

	c.CodecProperties = make([]byte, codecPropertiesLength)

	_, err = wire.Read(c.CodecProperties)
	if err != nil {
		return err
	}

	return nil
}

type BitmapCodecsCapabilitySet struct {
	BitmapCodecArray []BitmapCodec
}

func (s *BitmapCodecsCapabilitySet) Deserialize(wire io.Reader) error {
	var (
		bitmapCodecCount uint8
		err              error
	)

	err = binary.Read(wire, binary.LittleEndian, &bitmapCodecCount)
	if err != nil {
		return err
	}

	s.BitmapCodecArray = make([]BitmapCodec, bitmapCodecCount)

	for i := range s.BitmapCodecArray {
		err = s.BitmapCodecArray[i].Deserialize(wire)
		if err != nil {
			return err
		}
	}

	return nil
}

// NSCodec GUID: CA8D1BB9-000F-154F-589F-AE2D1A87E2D6
// Stored in little-endian format as per MS-RDPBCGR
var NSCodecGUID = [16]byte{
	0xB9, 0x1B, 0x8D, 0xCA, 0x0F, 0x00, 0x4F, 0x15,
	0x58, 0x9F, 0xAE, 0x2D, 0x1A, 0x87, 0xE2, 0xD6,
}

// NSCodecCapabilitySet represents the NSCodec-specific properties
type NSCodecCapabilitySet struct {
	FAllowDynamicFidelity uint8
	FAllowSubsampling     uint8
	ColorLossLevel        uint8
}

func (c *NSCodecCapabilitySet) Serialize() []byte {
	return []byte{
		c.FAllowDynamicFidelity,
		c.FAllowSubsampling,
		c.ColorLossLevel,
	}
}

func (c *BitmapCodec) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, c.CodecGUID)
	_ = binary.Write(buf, binary.LittleEndian, c.CodecID)
	_ = binary.Write(buf, binary.LittleEndian, uint16(len(c.CodecProperties)))
	buf.Write(c.CodecProperties)

	return buf.Bytes()
}

func (s *BitmapCodecsCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, uint8(len(s.BitmapCodecArray)))

	for _, codec := range s.BitmapCodecArray {
		buf.Write(codec.Serialize())
	}

	return buf.Bytes()
}

// NewBitmapCodecsCapabilitySet creates a capability set advertising NSCodec support
func NewBitmapCodecsCapabilitySet() CapabilitySet {
	nscodecProps := NSCodecCapabilitySet{
		FAllowDynamicFidelity: 1, // Allow dynamic fidelity
		FAllowSubsampling:     1, // Allow chroma subsampling
		ColorLossLevel:        3, // Moderate color loss (1=lossless, 7=max loss)
	}

	return CapabilitySet{
		CapabilitySetType: CapabilitySetTypeBitmapCodecs,
		BitmapCodecsCapabilitySet: &BitmapCodecsCapabilitySet{
			BitmapCodecArray: []BitmapCodec{
				{
					CodecGUID:       NSCodecGUID,
					CodecID:         1, // Will be assigned by server
					CodecProperties: nscodecProps.Serialize(),
				},
			},
		},
	}
}

type RailCapabilitySet struct {
	RailSupportLevel uint32
}

func NewRailCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType: CapabilitySetTypeRail,
		RailCapabilitySet: &RailCapabilitySet{
			RailSupportLevel: 1, // TS_RAIL_LEVEL_SUPPORTED
		},
	}
}

func (s *RailCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.RailSupportLevel)

	return buf.Bytes()
}

type WindowListCapabilitySet struct {
	WndSupportLevel     uint32
	NumIconCaches       uint8
	NumIconCacheEntries uint16
}

func NewWindowListCapabilitySet() CapabilitySet {
	return CapabilitySet{
		CapabilitySetType: CapabilitySetTypeWindow,
		WindowListCapabilitySet: &WindowListCapabilitySet{
			WndSupportLevel: 0, // TS_WINDOW_LEVEL_NOT_SUPPORTED
		},
	}
}

func (s *WindowListCapabilitySet) Serialize() []byte {
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.LittleEndian, s.WndSupportLevel)
	_ = binary.Write(buf, binary.LittleEndian, s.NumIconCaches)
	_ = binary.Write(buf, binary.LittleEndian, s.NumIconCacheEntries)

	return buf.Bytes()
}
