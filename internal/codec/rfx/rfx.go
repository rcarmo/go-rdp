// Package rfx implements the RemoteFX (RFX) codec decoder as specified in MS-RDPRFX.
// RemoteFX is a tile-based wavelet codec used for efficient remote desktop graphics.
package rfx

import "errors"

// Tile dimensions (fixed by MS-RDPRFX specification)
const (
	TileSize     = 64
	TilePixels   = TileSize * TileSize // 4096
	TileRGBASize = TilePixels * 4      // 16384 bytes
)

// Subband buffer offsets (linear/packed layout as used by FreeRDP)
// Coefficients are stored in this order in a 4096-element buffer:
// - HL1 (32×32 = 1024) at offset 0
// - LH1 (32×32 = 1024) at offset 1024
// - HH1 (32×32 = 1024) at offset 2048
// - HL2 (16×16 = 256)  at offset 3072
// - LH2 (16×16 = 256)  at offset 3328
// - HH2 (16×16 = 256)  at offset 3584
// - HL3 (8×8 = 64)     at offset 3840
// - LH3 (8×8 = 64)     at offset 3904
// - HH3 (8×8 = 64)     at offset 3968
// - LL3 (8×8 = 64)     at offset 4032
const (
	OffsetHL1 = 0
	OffsetLH1 = 1024
	OffsetHH1 = 2048
	OffsetHL2 = 3072
	OffsetLH2 = 3328
	OffsetHH2 = 3584
	OffsetHL3 = 3840
	OffsetLH3 = 3904
	OffsetHH3 = 3968
	OffsetLL3 = 4032

	SizeL1 = 1024 // 32×32
	SizeL2 = 256  // 16×16
	SizeL3 = 64   // 8×8
)

// RLGR coding modes
const (
	RLGR1 = 1 // Used for Y (luminance) component
	RLGR3 = 3 // Used for Cb, Cr (chrominance) components
)

// RLGR adaptive coding constants (from MS-RDPRFX section 3.1.8.1.7.1)
const (
	KPMAX = 80 // Maximum value for kp parameter
	LSGR  = 3  // Log2 scale factor for Golomb-Rice parameter
	UP_GR = 4  // Increment for run-length coding
	DN_GR = 6  // Decrement after non-zero value
	UQ_GR = 3  // Increment for zero value (RLGR1)
	DQ_GR = 3  // Decrement for non-zero value (RLGR1)
)

// Block type constants (MS-RDPRFX section 2.2.2.1.1)
const (
	WBT_SYNC           uint16 = 0xCCC0
	WBT_CODEC_VERSIONS uint16 = 0xCCC1
	WBT_CHANNELS       uint16 = 0xCCC2
	WBT_CONTEXT        uint16 = 0xCCC3
	WBT_FRAME_BEGIN    uint16 = 0xCCC4
	WBT_FRAME_END      uint16 = 0xCCC5
	WBT_REGION         uint16 = 0xCCC6
	WBT_EXTENSION      uint16 = 0xCCC7
	WBT_TILESET        uint16 = 0xCAC2
	CBT_TILE           uint16 = 0xCAC3
)

// Codec capability constants
const (
	CLW_VERSION_1_0    uint16 = 0x0100
	CT_TILE_64x64      uint16 = 0x0040
	CLW_COL_CONV_ICT   uint8  = 0x01
	CLW_XFORM_DWT_53_A uint8  = 0x01
	CLW_ENTROPY_RLGR1  uint8  = 0x01
	CLW_ENTROPY_RLGR3  uint8  = 0x04
)

// Errors
var (
	ErrInvalidBlockType   = errors.New("rfx: invalid block type")
	ErrInvalidBlockLength = errors.New("rfx: invalid block length")
	ErrInvalidTileData    = errors.New("rfx: invalid tile data")
	ErrRLGRDecodeError    = errors.New("rfx: RLGR decode error")
	ErrBufferTooSmall     = errors.New("rfx: buffer too small")
	ErrInvalidQuantValues = errors.New("rfx: invalid quantization values")
)

// SubbandQuant holds quantization values for all 10 subbands.
// Values are packed as 4-bit nibbles in the protocol.
type SubbandQuant struct {
	LL3 uint8 // Level 3 approximation (DC)
	LH3 uint8 // Level 3 horizontal detail
	HL3 uint8 // Level 3 vertical detail
	HH3 uint8 // Level 3 diagonal detail
	LH2 uint8 // Level 2 horizontal detail
	HL2 uint8 // Level 2 vertical detail
	HH2 uint8 // Level 2 diagonal detail
	LH1 uint8 // Level 1 horizontal detail
	HL1 uint8 // Level 1 vertical detail
	HH1 uint8 // Level 1 diagonal detail
}

// Tile represents a decoded 64×64 pixel tile
type Tile struct {
	X    uint16 // Tile X index (multiply by 64 for pixel position)
	Y    uint16 // Tile Y index (multiply by 64 for pixel position)
	RGBA []byte // Decoded RGBA pixels (16384 bytes)
}

// Frame represents a complete RFX frame with multiple tiles
type Frame struct {
	FrameIdx uint32
	Tiles    []*Tile
	Rects    []Rect
}

// Rect represents a rectangular region
type Rect struct {
	X, Y          uint16
	Width, Height uint16
}

// Context holds decoder state across frames
type Context struct {
	Width       uint16
	Height      uint16
	EntropyMode uint8 // RLGR1 or RLGR3

	// Quantization tables (indexed by quantIdxY, quantIdxCb, quantIdxCr)
	QuantTables []SubbandQuant
}

// NewContext creates a new RFX decoding context
func NewContext() *Context {
	return &Context{
		QuantTables: make([]SubbandQuant, 0, 8),
	}
}

// DefaultQuant returns default quantization values (quality ~85%)
func DefaultQuant() *SubbandQuant {
	return &SubbandQuant{
		LL3: 6, LH3: 6, HL3: 6, HH3: 6,
		LH2: 7, HL2: 7, HH2: 8,
		LH1: 8, HL1: 8, HH1: 9,
	}
}

// ParseQuantValues parses packed quantization values from the protocol.
// The quant values are packed as pairs of 4-bit nibbles:
// Byte 0: LL3 (low nibble), LH3 (high nibble)
// Byte 1: HL3 (low nibble), HH3 (high nibble)
// Byte 2: LH2 (low nibble), HL2 (high nibble)
// Byte 3: HH2 (low nibble), LH1 (high nibble)
// Byte 4: HL1 (low nibble), HH1 (high nibble)
func ParseQuantValues(data []byte) (*SubbandQuant, error) {
	if len(data) < 5 {
		return nil, ErrInvalidQuantValues
	}

	return &SubbandQuant{
		LL3: data[0] & 0x0F,
		LH3: (data[0] >> 4) & 0x0F,
		HL3: data[1] & 0x0F,
		HH3: (data[1] >> 4) & 0x0F,
		LH2: data[2] & 0x0F,
		HL2: (data[2] >> 4) & 0x0F,
		HH2: data[3] & 0x0F,
		LH1: (data[3] >> 4) & 0x0F,
		HL1: data[4] & 0x0F,
		HH1: (data[4] >> 4) & 0x0F,
	}, nil
}
