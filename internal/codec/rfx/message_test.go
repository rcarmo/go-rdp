package rfx

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRFXMessage_TooShort(t *testing.T) {
	ctx := NewContext()
	_, err := ParseRFXMessage([]byte{0x00, 0x01}, ctx)
	assert.Error(t, err)
}

func TestParseRFXMessage_InvalidBlockLength(t *testing.T) {
	ctx := NewContext()
	// Block with length 5 (less than minimum 6)
	data := []byte{
		0xC0, 0xCC, // WBT_SYNC
		0x05, 0x00, 0x00, 0x00, // length = 5 (invalid)
	}
	_, err := ParseRFXMessage(data, ctx)
	assert.Error(t, err)
}

func TestParseRFXMessage_SyncBlock(t *testing.T) {
	ctx := NewContext()
	// Valid SYNC block
	data := []byte{
		0xC0, 0xCC, // WBT_SYNC
		0x0C, 0x00, 0x00, 0x00, // length = 12
		0xCA, 0xCA, 0xCC, 0xCA, // magic
		0x00, 0x01, // version
	}
	frame, err := ParseRFXMessage(data, ctx)
	require.NoError(t, err)
	assert.NotNil(t, frame)
}

func TestParseRFXMessage_ContextBlock(t *testing.T) {
	ctx := NewContext()
	// Valid CONTEXT block
	data := []byte{
		0xC3, 0xCC, // WBT_CONTEXT
		0x0D, 0x00, 0x00, 0x00, // length = 13
		0x00,       // ctxId
		0x40, 0x00, // tileSize = 64
		0x00, 0x04, // width = 1024
		0x00, 0x03, // height = 768
	}
	frame, err := ParseRFXMessage(data, ctx)
	require.NoError(t, err)
	assert.NotNil(t, frame)
	assert.Equal(t, uint16(1024), ctx.Width)
	assert.Equal(t, uint16(768), ctx.Height)
}

func TestParseRFXMessage_FrameBegin(t *testing.T) {
	ctx := NewContext()
	// Valid FRAME_BEGIN block
	data := []byte{
		0xC4, 0xCC, // WBT_FRAME_BEGIN
		0x0E, 0x00, 0x00, 0x00, // length = 14
		0x05, 0x00, 0x00, 0x00, // frameIdx = 5
		0x01, 0x00, // numRegions = 1
		0x00, 0x00, // padding
	}
	frame, err := ParseRFXMessage(data, ctx)
	require.NoError(t, err)
	assert.NotNil(t, frame)
	assert.Equal(t, uint32(5), frame.FrameIdx)
}

func TestParseRFXMessage_RegionBlock(t *testing.T) {
	ctx := NewContext()
	// Valid REGION block with 1 rect
	data := []byte{
		0xC6, 0xCC, // WBT_REGION
		0x11, 0x00, 0x00, 0x00, // length = 17
		0x00,       // regionFlags
		0x01, 0x00, // numRects = 1
		// Rect 1
		0x10, 0x00, // x = 16
		0x20, 0x00, // y = 32
		0x40, 0x00, // width = 64
		0x40, 0x00, // height = 64
	}
	frame, err := ParseRFXMessage(data, ctx)
	require.NoError(t, err)
	require.Len(t, frame.Rects, 1)
	assert.Equal(t, uint16(16), frame.Rects[0].X)
	assert.Equal(t, uint16(32), frame.Rects[0].Y)
	assert.Equal(t, uint16(64), frame.Rects[0].Width)
	assert.Equal(t, uint16(64), frame.Rects[0].Height)
}

func TestParseRFXMessage_RegionBlockMultipleRects(t *testing.T) {
	ctx := NewContext()
	// Valid REGION block with 2 rects
	data := []byte{
		0xC6, 0xCC, // WBT_REGION
		0x19, 0x00, 0x00, 0x00, // length = 25
		0x00,       // regionFlags
		0x02, 0x00, // numRects = 2
		// Rect 1
		0x00, 0x00, // x = 0
		0x00, 0x00, // y = 0
		0x40, 0x00, // width = 64
		0x40, 0x00, // height = 64
		// Rect 2
		0x40, 0x00, // x = 64
		0x00, 0x00, // y = 0
		0x40, 0x00, // width = 64
		0x40, 0x00, // height = 64
	}
	frame, err := ParseRFXMessage(data, ctx)
	require.NoError(t, err)
	require.Len(t, frame.Rects, 2)
	assert.Equal(t, uint16(0), frame.Rects[0].X)
	assert.Equal(t, uint16(64), frame.Rects[1].X)
}

func TestParseRFXMessage_FrameEnd(t *testing.T) {
	ctx := NewContext()
	// Valid FRAME_END block
	data := []byte{
		0xC5, 0xCC, // WBT_FRAME_END
		0x06, 0x00, 0x00, 0x00, // length = 6 (just header)
	}
	frame, err := ParseRFXMessage(data, ctx)
	require.NoError(t, err)
	assert.NotNil(t, frame)
}

func TestParseRFXMessage_CodecVersions(t *testing.T) {
	ctx := NewContext()
	// Valid CODEC_VERSIONS block
	data := []byte{
		0xC1, 0xCC, // WBT_CODEC_VERSIONS
		0x0A, 0x00, 0x00, 0x00, // length = 10
		0x01,       // numCodecs
		0x01,       // codecId
		0x00, 0x01, // version
	}
	frame, err := ParseRFXMessage(data, ctx)
	require.NoError(t, err)
	assert.NotNil(t, frame)
}

func TestParseRFXMessage_Channels(t *testing.T) {
	ctx := NewContext()
	// Valid CHANNELS block
	data := []byte{
		0xC2, 0xCC, // WBT_CHANNELS
		0x0C, 0x00, 0x00, 0x00, // length = 12
		0x01,       // numChannels
		0x00,       // channelId
		0x00, 0x04, // width
		0x00, 0x03, // height
	}
	frame, err := ParseRFXMessage(data, ctx)
	require.NoError(t, err)
	assert.NotNil(t, frame)
}

func TestParseRFXMessage_Extension(t *testing.T) {
	ctx := NewContext()
	// Valid EXTENSION block (should be skipped)
	data := []byte{
		0xC7, 0xCC, // WBT_EXTENSION
		0x0A, 0x00, 0x00, 0x00, // length = 10
		0x00, 0x00, 0x00, 0x00, // extension data
	}
	frame, err := ParseRFXMessage(data, ctx)
	require.NoError(t, err)
	assert.NotNil(t, frame)
}

func TestParseSyncBlock_TooShort(t *testing.T) {
	err := parseSyncBlock([]byte{0x00, 0x01, 0x02})
	assert.Error(t, err)
}

func TestParseContextBlock_TooShort(t *testing.T) {
	ctx := NewContext()
	err := parseContextBlock([]byte{0x00, 0x01, 0x02}, ctx)
	assert.Error(t, err)
}

func TestParseFrameBegin_TooShort(t *testing.T) {
	_, err := parseFrameBegin([]byte{0x00, 0x01, 0x02})
	assert.Error(t, err)
}

func TestParseRegionBlock_TooShort(t *testing.T) {
	_, err := parseRegionBlock([]byte{0x00, 0x01, 0x02})
	assert.Error(t, err)
}

func TestParseTilesetBlock_TooShort(t *testing.T) {
	ctx := NewContext()
	_, err := parseTilesetBlock([]byte{0x00, 0x01, 0x02}, ctx)
	assert.Error(t, err)
}

func TestParseTilesetBlock_WithQuantTables(t *testing.T) {
	ctx := NewContext()
	
	// Build a minimal tileset block with 1 quant table and 0 tiles
	data := make([]byte, 27)
	binary.LittleEndian.PutUint16(data[0:], WBT_TILESET) // blockType
	binary.LittleEndian.PutUint32(data[2:], 27)         // blockLen
	binary.LittleEndian.PutUint16(data[6:], 0x0001)     // subtype
	binary.LittleEndian.PutUint16(data[8:], 0x0000)     // idx
	binary.LittleEndian.PutUint16(data[10:], 0x0000)    // flags
	data[12] = 1                                         // numQuant = 1
	data[13] = 64                                        // tileSize = 64
	binary.LittleEndian.PutUint16(data[14:], 0)         // numTiles = 0
	binary.LittleEndian.PutUint32(data[16:], 0)         // tileDataSize
	// Quant table (5 bytes)
	data[20] = 0x66 // LL3=6, LH3=6
	data[21] = 0x66 // HL3=6, HH3=6
	data[22] = 0x77 // LH2=7, HL2=7
	data[23] = 0x88 // HH2=8, LH1=8
	data[24] = 0x99 // HL1=9, HH1=9
	// 2 padding bytes to reach 27
	data[25] = 0x00
	data[26] = 0x00

	tiles, err := parseTilesetBlock(data, ctx)
	require.NoError(t, err)
	assert.Empty(t, tiles)
}

func TestParseRFXMessage_MultipleBlocks(t *testing.T) {
	ctx := NewContext()
	
	// Build a message with multiple blocks
	data := []byte{}
	
	// SYNC block
	syncBlock := []byte{
		0xC0, 0xCC, // WBT_SYNC
		0x0C, 0x00, 0x00, 0x00, // length = 12
		0xCA, 0xCA, 0xCC, 0xCA, // magic
		0x00, 0x01, // version
	}
	data = append(data, syncBlock...)
	
	// CONTEXT block
	contextBlock := []byte{
		0xC3, 0xCC, // WBT_CONTEXT
		0x0D, 0x00, 0x00, 0x00, // length = 13
		0x00,       // ctxId
		0x40, 0x00, // tileSize = 64
		0x80, 0x02, // width = 640
		0xE0, 0x01, // height = 480
	}
	data = append(data, contextBlock...)
	
	// FRAME_BEGIN block
	frameBeginBlock := []byte{
		0xC4, 0xCC, // WBT_FRAME_BEGIN
		0x0E, 0x00, 0x00, 0x00, // length = 14
		0x01, 0x00, 0x00, 0x00, // frameIdx = 1
		0x01, 0x00, // numRegions = 1
		0x00, 0x00, // padding
	}
	data = append(data, frameBeginBlock...)
	
	// FRAME_END block
	frameEndBlock := []byte{
		0xC5, 0xCC, // WBT_FRAME_END
		0x06, 0x00, 0x00, 0x00, // length = 6
	}
	data = append(data, frameEndBlock...)
	
	frame, err := ParseRFXMessage(data, ctx)
	require.NoError(t, err)
	assert.NotNil(t, frame)
	assert.Equal(t, uint16(640), ctx.Width)
	assert.Equal(t, uint16(480), ctx.Height)
	assert.Equal(t, uint32(1), frame.FrameIdx)
}

func TestParseRFXMessage_PartialBlock(t *testing.T) {
	ctx := NewContext()
	// Block header present but data truncated (length says 12 but only 8 bytes)
	data := []byte{
		0xC0, 0xCC, // WBT_SYNC
		0x0C, 0x00, 0x00, 0x00, // length = 12
		0xCA, 0xCA, // only 2 data bytes instead of 6
	}
	_, err := ParseRFXMessage(data, ctx)
	assert.Error(t, err)
}
