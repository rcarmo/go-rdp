package rfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeTile_TooShort(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02, 0x03}
	quant := DefaultQuant()

	tile, err := DecodeTile(data, quant, quant, quant)
	assert.Error(t, err)
	assert.Nil(t, tile)
}

func TestDecodeTile_WrongBlockType(t *testing.T) {
	// Create data with wrong block type
	data := make([]byte, 50)
	data[0] = 0x00 // Wrong block type (not 0xCAC3)
	data[1] = 0x00

	quant := DefaultQuant()

	tile, err := DecodeTile(data, quant, quant, quant)
	assert.Equal(t, ErrInvalidBlockType, err)
	assert.Nil(t, tile)
}

func TestDecodeTile_InvalidBlockLength(t *testing.T) {
	// Block length larger than data
	data := []byte{
		0xC3, 0xCA, // Block type = 0xCAC3
		0xFF, 0x00, 0x00, 0x00, // Block length = 255 (larger than data)
		0x00, 0x00, 0x00, // Quant indices
		0x00, 0x00, // X index
		0x00, 0x00, // Y index
		0x00, 0x00, // Y data length
		0x00, 0x00, // Cb data length
		0x00, 0x00, // Cr data length
	}

	quant := DefaultQuant()
	tile, err := DecodeTile(data, quant, quant, quant)
	assert.Equal(t, ErrInvalidBlockLength, err)
	assert.Nil(t, tile)
}

func TestDecodeTile_InvalidComponentLengths(t *testing.T) {
	// Component lengths larger than remaining data
	data := []byte{
		0xC3, 0xCA, // Block type = 0xCAC3
		0x13, 0x00, 0x00, 0x00, // Block length = 19
		0x00, 0x00, 0x00, // Quant indices
		0x00, 0x00, // X index
		0x00, 0x00, // Y index
		0x10, 0x00, // Y data length = 16 (too long)
		0x00, 0x00, // Cb data length
		0x00, 0x00, // Cr data length
	}

	quant := DefaultQuant()
	tile, err := DecodeTile(data, quant, quant, quant)
	assert.Equal(t, ErrInvalidTileData, err)
	assert.Nil(t, tile)
}

func TestDecodeTile_ValidMinimalTile(t *testing.T) {
	// Create a minimal valid tile structure
	// Block type: CBT_TILE (0xCAC3)
	// Block length: 19 (header only, no component data)
	data := []byte{
		0xC3, 0xCA, // Block type = 0xCAC3
		0x13, 0x00, 0x00, 0x00, // Block length = 19
		0x00, 0x00, 0x00, // Quant indices (Y, Cb, Cr)
		0x01, 0x00, // X index = 1
		0x02, 0x00, // Y index = 2
		0x00, 0x00, // Y data length = 0
		0x00, 0x00, // Cb data length = 0
		0x00, 0x00, // Cr data length = 0
	}

	quant := DefaultQuant()

	tile, err := DecodeTile(data, quant, quant, quant)
	require.NoError(t, err)
	require.NotNil(t, tile)

	assert.Equal(t, uint16(1), tile.X)
	assert.Equal(t, uint16(2), tile.Y)
	assert.Len(t, tile.RGBA, TileRGBASize)
}

func TestDecodeTileWithBuffers_TooShort(t *testing.T) {
	data := []byte{0x00, 0x01, 0x02}
	quant := DefaultQuant()

	yCoeff := make([]int16, TilePixels)
	cbCoeff := make([]int16, TilePixels)
	crCoeff := make([]int16, TilePixels)
	rgba := make([]byte, TileRGBASize)

	_, _, err := DecodeTileWithBuffers(data, quant, quant, quant, yCoeff, cbCoeff, crCoeff, rgba)
	assert.Error(t, err)
}

func TestDecodeTileWithBuffers_WrongBlockType(t *testing.T) {
	data := make([]byte, 50)
	data[0] = 0x00 // Wrong block type
	data[1] = 0x00

	quant := DefaultQuant()
	yCoeff := make([]int16, TilePixels)
	cbCoeff := make([]int16, TilePixels)
	crCoeff := make([]int16, TilePixels)
	rgba := make([]byte, TileRGBASize)

	_, _, err := DecodeTileWithBuffers(data, quant, quant, quant, yCoeff, cbCoeff, crCoeff, rgba)
	assert.Equal(t, ErrInvalidBlockType, err)
}

func TestDecodeTileWithBuffers_InvalidBlockLength(t *testing.T) {
	data := []byte{
		0xC3, 0xCA, // Block type = 0xCAC3
		0xFF, 0x00, 0x00, 0x00, // Block length = 255 (larger than data)
		0x00, 0x00, 0x00, // Quant indices
		0x00, 0x00, // X index
		0x00, 0x00, // Y index
		0x00, 0x00, // Y data length
		0x00, 0x00, // Cb data length
		0x00, 0x00, // Cr data length
	}

	quant := DefaultQuant()
	yCoeff := make([]int16, TilePixels)
	cbCoeff := make([]int16, TilePixels)
	crCoeff := make([]int16, TilePixels)
	rgba := make([]byte, TileRGBASize)

	_, _, err := DecodeTileWithBuffers(data, quant, quant, quant, yCoeff, cbCoeff, crCoeff, rgba)
	assert.Equal(t, ErrInvalidBlockLength, err)
}

func TestDecodeTileWithBuffers_InvalidComponentLengths(t *testing.T) {
	data := []byte{
		0xC3, 0xCA, // Block type = 0xCAC3
		0x13, 0x00, 0x00, 0x00, // Block length = 19
		0x00, 0x00, 0x00, // Quant indices
		0x00, 0x00, // X index
		0x00, 0x00, // Y index
		0x10, 0x00, // Y data length = 16 (too long)
		0x00, 0x00, // Cb data length
		0x00, 0x00, // Cr data length
	}

	quant := DefaultQuant()
	yCoeff := make([]int16, TilePixels)
	cbCoeff := make([]int16, TilePixels)
	crCoeff := make([]int16, TilePixels)
	rgba := make([]byte, TileRGBASize)

	_, _, err := DecodeTileWithBuffers(data, quant, quant, quant, yCoeff, cbCoeff, crCoeff, rgba)
	assert.Equal(t, ErrInvalidTileData, err)
}

func TestDecodeTileWithBuffers_ValidMinimalTile(t *testing.T) {
	data := []byte{
		0xC3, 0xCA, // Block type = 0xCAC3
		0x13, 0x00, 0x00, 0x00, // Block length = 19
		0x00, 0x00, 0x00, // Quant indices
		0x03, 0x00, // X index = 3
		0x04, 0x00, // Y index = 4
		0x00, 0x00, // Y data length = 0
		0x00, 0x00, // Cb data length = 0
		0x00, 0x00, // Cr data length = 0
	}

	quant := DefaultQuant()
	yCoeff := make([]int16, TilePixels)
	cbCoeff := make([]int16, TilePixels)
	crCoeff := make([]int16, TilePixels)
	rgba := make([]byte, TileRGBASize)

	xIdx, yIdx, err := DecodeTileWithBuffers(data, quant, quant, quant, yCoeff, cbCoeff, crCoeff, rgba)
	require.NoError(t, err)
	assert.Equal(t, uint16(3), xIdx)
	assert.Equal(t, uint16(4), yIdx)
}

func TestNewContext(t *testing.T) {
	ctx := NewContext()
	require.NotNil(t, ctx)
	assert.NotNil(t, ctx.QuantTables)
	assert.Empty(t, ctx.QuantTables)
}

func TestTileConstants(t *testing.T) {
	assert.Equal(t, 64, TileSize)
	assert.Equal(t, 4096, TilePixels)
	assert.Equal(t, 16384, TileRGBASize)
}

func TestSubbandOffsetConstants(t *testing.T) {
	assert.Equal(t, 0, OffsetHL1)
	assert.Equal(t, 1024, OffsetLH1)
	assert.Equal(t, 2048, OffsetHH1)
	assert.Equal(t, 3072, OffsetHL2)
	assert.Equal(t, 3328, OffsetLH2)
	assert.Equal(t, 3584, OffsetHH2)
	assert.Equal(t, 3840, OffsetHL3)
	assert.Equal(t, 3904, OffsetLH3)
	assert.Equal(t, 3968, OffsetHH3)
	assert.Equal(t, 4032, OffsetLL3)

	// Verify sizes
	assert.Equal(t, 1024, SizeL1)
	assert.Equal(t, 256, SizeL2)
	assert.Equal(t, 64, SizeL3)

	// Verify total = 4096
	total := SizeL1*3 + SizeL2*3 + SizeL3*4
	assert.Equal(t, TilePixels, total)
}

func TestRLGRModeConstants(t *testing.T) {
	assert.Equal(t, 1, RLGR1)
	assert.Equal(t, 3, RLGR3)
}

func TestBlockTypeConstants(t *testing.T) {
	assert.Equal(t, uint16(0xCCC0), WBT_SYNC)
	assert.Equal(t, uint16(0xCCC1), WBT_CODEC_VERSIONS)
	assert.Equal(t, uint16(0xCCC2), WBT_CHANNELS)
	assert.Equal(t, uint16(0xCCC3), WBT_CONTEXT)
	assert.Equal(t, uint16(0xCCC4), WBT_FRAME_BEGIN)
	assert.Equal(t, uint16(0xCCC5), WBT_FRAME_END)
	assert.Equal(t, uint16(0xCCC6), WBT_REGION)
	assert.Equal(t, uint16(0xCAC2), WBT_TILESET)
	assert.Equal(t, uint16(0xCAC3), CBT_TILE)
}
