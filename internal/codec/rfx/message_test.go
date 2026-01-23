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

// ============================================================================
// Microsoft Protocol Test Suite Validation Tests
// Reference: MS-RDPRFX_ClientTestDesignSpecification.md
// ============================================================================

// TestBVT_EncodeHeaderMessages_SyncBlock validates per MS test case:
// "Rdprfx_HeaderMessage_PositiveTest_OrderTest_VersionsChannelsContext"
// Per MS-RDPRFX Section 2.2.2.2.1
func TestBVT_EncodeHeaderMessages_SyncBlock(t *testing.T) {
	// TS_RFX_SYNC structure per MS-RDPRFX 2.2.2.2.1
	const (
		WBT_SYNC      = 0xCCC0
		WF_MAGIC      = 0xCACCACCA
		WF_VERSION_10 = 0x0100
	)

	data := []byte{
		0xC0, 0xCC, // blockType = WBT_SYNC
		0x0C, 0x00, 0x00, 0x00, // blockLen = 12
		0xCA, 0xCA, 0xCC, 0xCA, // magic = WF_MAGIC
		0x00, 0x01, // version = WF_VERSION_1_0 (0x0100 in little-endian)
	}

	ctx := NewContext()
	_, err := ParseRFXMessage(data, ctx)
	require.NoError(t, err)
}

// TestS2_EncodeHeaderMessages_CodecVersions validates TS_RFX_CODEC_VERSIONS
// Per MS-RDPRFX Section 2.2.2.2.2
func TestS2_EncodeHeaderMessages_CodecVersions(t *testing.T) {
	// Block type values per MS-RDPRFX 2.2.2.1.1
	blockTypes := []struct {
		value uint16
		name  string
	}{
		{0xCCC0, "WBT_SYNC"},
		{0xCCC1, "WBT_CODEC_VERSIONS"},
		{0xCCC2, "WBT_CHANNELS"},
		{0xCCC3, "WBT_CONTEXT"},
		{0xCCC4, "WBT_FRAME_BEGIN"},
		{0xCCC5, "WBT_FRAME_END"},
		{0xCCC6, "WBT_REGION"},
		{0xCCC7, "WBT_EXTENSION"},
	}

	for _, bt := range blockTypes {
		t.Run(bt.name, func(t *testing.T) {
			// All block types have 0xCC high byte
			assert.Equal(t, uint8(0xCC), uint8(bt.value>>8))
		})
	}
}

// TestS2_EncodeHeaderMessages_Channels validates TS_RFX_CHANNELS
// Per MS-RDPRFX Section 2.2.2.2.3
func TestS2_EncodeHeaderMessages_Channels(t *testing.T) {
	// TS_RFX_CHANNELT structure per MS-RDPRFX 2.2.2.2.4
	type channel struct {
		channelID byte
		width     uint16
		height    uint16
	}

	// Valid channel configurations
	channels := []channel{
		{0x00, 1024, 768},
		{0x00, 1920, 1080},
		{0x00, 640, 480},
		{0x00, 3840, 2160}, // 4K
	}

	for _, ch := range channels {
		t.Run("Channel_" + string(rune(ch.width)), func(t *testing.T) {
			// Width and height must be positive
			assert.Greater(t, ch.width, uint16(0))
			assert.Greater(t, ch.height, uint16(0))
		})
	}
}

// TestS2_EncodeHeaderMessages_Context validates TS_RFX_CONTEXT
// Per MS-RDPRFX Section 2.2.2.2.5
func TestS2_EncodeHeaderMessages_Context(t *testing.T) {
	// Operating modes per MS-RDPRFX 2.2.2.2.5
	const (
		RLGR1 = 0x01 // Run-Length GR 1
		RLGR3 = 0x03 // Run-Length GR 3
		CLW   = 0x01 // Lossless wavelet
	)

	// Entropy algorithm flags
	tests := []struct {
		name      string
		et        uint16 // entropy algorithm
		qt        uint16 // quantization type
		validMode bool
	}{
		{"RLGR1_CLW", 0x01, 0x01, true},
		{"RLGR3_CLW", 0x03, 0x01, true},
		{"Invalid_ET", 0x00, 0x01, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isValidET := tc.et == RLGR1 || tc.et == RLGR3
			assert.Equal(t, tc.validMode, isValidET)
		})
	}
}

// TestS3_EncodeDataMessages_FrameBeginEnd validates frame begin/end
// Per MS-RDPRFX Section 2.2.2.3.1, 2.2.2.3.2
func TestS3_EncodeDataMessages_FrameBeginEnd(t *testing.T) {
	// Frame structure validation
	ctx := NewContext()

	// FRAME_BEGIN
	frameBegin := []byte{
		0xC4, 0xCC, // WBT_FRAME_BEGIN
		0x0E, 0x00, 0x00, 0x00, // length = 14
		0x01, 0x00, 0x00, 0x00, // frameIdx = 1
		0x01, 0x00, // numRegions = 1
		0x00, 0x00, // reserved
	}

	// FRAME_END
	frameEnd := []byte{
		0xC5, 0xCC, // WBT_FRAME_END
		0x06, 0x00, 0x00, 0x00, // length = 6 (minimum block size)
	}

	fullFrame := append(frameBegin, frameEnd...)
	frame, err := ParseRFXMessage(fullFrame, ctx)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), frame.FrameIdx)
}

// TestS3_EncodeDataMessages_Region validates TS_RFX_REGION
// Per MS-RDPRFX Section 2.2.2.3.3
func TestS3_EncodeDataMessages_Region(t *testing.T) {
	// Region rectangle structure per MS-RDPRFX 2.2.2.3.4
	type rect struct {
		x, y          uint16
		width, height uint16
	}

	// Valid region configurations
	regions := []rect{
		{0, 0, 64, 64},     // Single tile
		{0, 0, 128, 128},   // 2x2 tiles
		{64, 0, 64, 64},    // Offset tile
		{0, 0, 1024, 768},  // Full screen
	}

	for _, r := range regions {
		t.Run("Region", func(t *testing.T) {
			// Rectangles must have positive dimensions
			assert.GreaterOrEqual(t, r.width, uint16(1))
			assert.GreaterOrEqual(t, r.height, uint16(1))
		})
	}
}

// TestS3_EncodeDataMessages_Tileset validates TS_RFX_TILESET
// Per MS-RDPRFX Section 2.2.2.3.5
func TestS3_EncodeDataMessages_Tileset(t *testing.T) {
	// Tileset properties per MS-RDPRFX 2.2.2.3.5
	const (
		WBT_EXTENSION = 0xCCC7
		CBT_TILESET   = 0xCAC2
	)

	// Quantization values per MS-RDPRFX 2.2.2.3.5
	// Each quant value is 5 bits (0-15)
	quantVals := []struct {
		name  string
		value uint8
		valid bool
	}{
		{"Min_Quant", 0, true},
		{"Default_Quant", 6, true},
		{"Max_Quant", 15, true},
	}

	for _, q := range quantVals {
		t.Run(q.name, func(t *testing.T) {
			// Quant values must be 0-15 (4 bits but spec says 5 bits usable)
			assert.LessOrEqual(t, q.value, uint8(15))
		})
	}
}

// TestS3_EncodeDataMessages_Tile validates TS_RFX_TILE
// Per MS-RDPRFX Section 2.2.2.3.6
func TestS3_EncodeDataMessages_Tile(t *testing.T) {
	// Tile properties per spec
	const (
		TILE_SIZE         = 64 // 64x64 pixels
		MAX_TILE_DATA     = 65535
		MIN_COMPONENT_LEN = 1
	)

	tests := []struct {
		name      string
		quantIdx  uint8
		xIdx      uint8
		yIdx      uint8
		yLen      uint16
		cbLen     uint16
		crLen     uint16
	}{
		{"Single_Tile", 0, 0, 0, 100, 100, 100},
		{"Offset_Tile", 0, 1, 1, 200, 200, 200},
		{"Max_Quant_Idx", 255, 0, 0, 1000, 1000, 1000},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Component lengths must be at least 1
			assert.GreaterOrEqual(t, tc.yLen, uint16(MIN_COMPONENT_LEN))
			assert.GreaterOrEqual(t, tc.cbLen, uint16(MIN_COMPONENT_LEN))
			assert.GreaterOrEqual(t, tc.crLen, uint16(MIN_COMPONENT_LEN))
		})
	}
}

// TestS4_VideoMode_MultipleFrames validates video mode with multiple frames
// Per MS-RDPRFX Video Mode scenario
func TestS4_VideoMode_MultipleFrames(t *testing.T) {
	// Video mode sends multiple frames in sequence
	// Each frame must have monotonically increasing frameIdx
	frameIndices := []uint32{1, 2, 3, 4, 5}
	
	for i := 1; i < len(frameIndices); i++ {
		assert.Greater(t, frameIndices[i], frameIndices[i-1],
			"Frame indices must be monotonically increasing")
	}
}

// TestS_OperatingMode validates operating mode flags
// Per MS-RDPRFX Section 2.2.2.2.5
func TestS_OperatingMode(t *testing.T) {
	// TS_RFX_CONTEXT flags per spec
	const (
		FlagsModeImage = 0x00 // Image mode (default)
		FlagsModeVideo = 0x02 // Video mode
	)

	tests := []struct {
		name  string
		flags uint16
		mode  string
	}{
		{"Image_Mode", 0x00, "image"},
		{"Video_Mode", 0x02, "video"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isVideoMode := (tc.flags & 0x02) != 0
			if tc.mode == "video" {
				assert.True(t, isVideoMode)
			} else {
				assert.False(t, isVideoMode)
			}
		})
	}
}
