package rfx

import (
	"encoding/binary"
)

// DecodeTile decodes a single RFX tile from compressed data.
// data: raw tile data starting with CBT_TILE block header
// quantY, quantCb, quantCr: quantization values for each component
func DecodeTile(data []byte, quantY, quantCb, quantCr *SubbandQuant) (*Tile, error) {
	if len(data) < 19 { // Minimum tile header size
		return nil, ErrInvalidTileData
	}

	offset := 0

	// Parse block header
	blockType := binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	if blockType != CBT_TILE {
		return nil, ErrInvalidBlockType
	}

	blockLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	if int(blockLen) > len(data) {
		return nil, ErrInvalidBlockLength
	}

	// Parse tile header (MS-RDPRFX 2.2.4.4)
	// quantIdxY := data[offset]
	// quantIdxCb := data[offset+1]
	// quantIdxCr := data[offset+2]
	offset += 3

	xIdx := binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	yIdx := binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	// Component data sizes
	yLen := int(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2

	cbLen := int(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2

	crLen := int(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2

	// Validate lengths
	if offset+yLen+cbLen+crLen > len(data) {
		return nil, ErrInvalidTileData
	}

	// Extract component data
	yData := data[offset : offset+yLen]
	offset += yLen

	cbData := data[offset : offset+cbLen]
	offset += cbLen

	crData := data[offset : offset+crLen]

	// Allocate coefficient buffers
	yCoeff := make([]int16, TilePixels)
	cbCoeff := make([]int16, TilePixels)
	crCoeff := make([]int16, TilePixels)

	// RLGR decode each component
	if err := RLGRDecode(yData, RLGR1, yCoeff); err != nil {
		return nil, err
	}
	if err := RLGRDecode(cbData, RLGR3, cbCoeff); err != nil {
		return nil, err
	}
	if err := RLGRDecode(crData, RLGR3, crCoeff); err != nil {
		return nil, err
	}

	// Differential decode LL3 subband (DC coefficients)
	DifferentialDecode(yCoeff[OffsetLL3:], SizeL3)
	DifferentialDecode(cbCoeff[OffsetLL3:], SizeL3)
	DifferentialDecode(crCoeff[OffsetLL3:], SizeL3)

	// Dequantize
	Dequantize(yCoeff, quantY)
	Dequantize(cbCoeff, quantCb)
	Dequantize(crCoeff, quantCr)

	// Inverse DWT
	yPixels := InverseDWT2D(yCoeff)
	cbPixels := InverseDWT2D(cbCoeff)
	crPixels := InverseDWT2D(crCoeff)

	// Color convert to RGBA
	rgba := make([]byte, TileRGBASize)
	YCbCrToRGBA(yPixels, cbPixels, crPixels, rgba)

	return &Tile{
		X:    xIdx,
		Y:    yIdx,
		RGBA: rgba,
	}, nil
}

// DecodeTileWithBuffers decodes a tile using pre-allocated buffers (for WASM).
// This avoids allocations in the hot path.
func DecodeTileWithBuffers(
	data []byte,
	quantY, quantCb, quantCr *SubbandQuant,
	yCoeff, cbCoeff, crCoeff []int16,
	rgba []byte,
) (xIdx, yIdx uint16, err error) {
	if len(data) < 19 {
		return 0, 0, ErrInvalidTileData
	}

	offset := 0

	// Parse block header
	blockType := binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	if blockType != CBT_TILE {
		return 0, 0, ErrInvalidBlockType
	}

	blockLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	if int(blockLen) > len(data) {
		return 0, 0, ErrInvalidBlockLength
	}

	// Skip quant indices
	offset += 3

	xIdx = binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	yIdx = binary.LittleEndian.Uint16(data[offset:])
	offset += 2

	yLen := int(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2

	cbLen := int(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2

	crLen := int(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2

	if offset+yLen+cbLen+crLen > len(data) {
		return 0, 0, ErrInvalidTileData
	}

	// RLGR decode
	if err := RLGRDecode(data[offset:offset+yLen], RLGR1, yCoeff); err != nil {
		return 0, 0, err
	}
	offset += yLen

	if err := RLGRDecode(data[offset:offset+cbLen], RLGR3, cbCoeff); err != nil {
		return 0, 0, err
	}
	offset += cbLen

	if err := RLGRDecode(data[offset:offset+crLen], RLGR3, crCoeff); err != nil {
		return 0, 0, err
	}

	// Differential decode LL3 subband (DC coefficients)
	DifferentialDecode(yCoeff[OffsetLL3:], SizeL3)
	DifferentialDecode(cbCoeff[OffsetLL3:], SizeL3)
	DifferentialDecode(crCoeff[OffsetLL3:], SizeL3)

	// Dequantize
	Dequantize(yCoeff, quantY)
	Dequantize(cbCoeff, quantCb)
	Dequantize(crCoeff, quantCr)

	// Inverse DWT
	yPixels := InverseDWT2D(yCoeff)
	cbPixels := InverseDWT2D(cbCoeff)
	crPixels := InverseDWT2D(crCoeff)

	// Color convert
	YCbCrToRGBA(yPixels, cbPixels, crPixels, rgba)

	return xIdx, yIdx, nil
}
