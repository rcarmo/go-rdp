package codec

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoundUpToMultiple(t *testing.T) {
	tests := []struct {
		name     string
		n, m     int
		expected int
	}{
		{"zero value", 0, 8, 0},
		{"less than multiple", 1, 8, 8},
		{"one less than multiple", 7, 8, 8},
		{"exact multiple", 8, 8, 8},
		{"one more than multiple", 9, 8, 16},
		{"large value", 100, 8, 104},
		{"multiple of 2 less", 5, 2, 6},
		{"multiple of 2 exact", 6, 2, 6},
		{"zero multiple", 10, 0, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := roundUpToMultiple(tt.n, tt.m)
			assert.Equal(t, tt.expected, result, "roundUpToMultiple(%d, %d)", tt.n, tt.m)
		})
	}
}

func TestClamp(t *testing.T) {
	assert.Equal(t, 0, clamp(-10))
	assert.Equal(t, 0, clamp(0))
	assert.Equal(t, 128, clamp(128))
	assert.Equal(t, 255, clamp(255))
	assert.Equal(t, 255, clamp(300))
}

func TestRestoreColorLoss(t *testing.T) {
	// ColorLossLevel 1 should return unchanged
	input := []byte{10, 20, 30}
	result := restoreColorLoss(input, 1)
	assert.Equal(t, input, result)

	// ColorLossLevel 2 should shift by 1
	input = []byte{10, 20, 30}
	result = restoreColorLoss(input, 2)
	assert.Equal(t, []byte{20, 40, 60}, result)

	// ColorLossLevel 3 should shift by 2
	input = []byte{10, 20, 30}
	result = restoreColorLoss(input, 3)
	assert.Equal(t, []byte{40, 80, 120}, result)

	// Should clamp at 255
	input = []byte{200}
	result = restoreColorLoss(input, 3)
	assert.Equal(t, []byte{255}, result) // 200 << 2 = 800, clamped to 255
}

func TestChromaSuperSample(t *testing.T) {
	// 2x2 source -> 4x4 destination
	src := []byte{1, 2, 3, 4}
	result := chromaSuperSample(src, 2, 2, 4, 4)

	// Each source pixel should be replicated to 2x2 block
	expected := []byte{
		1, 1, 2, 2,
		1, 1, 2, 2,
		3, 3, 4, 4,
		3, 3, 4, 4,
	}
	assert.Equal(t, expected, result)
}

func TestAycoCgToRGBA(t *testing.T) {
	// Simple 1x1 pixel test
	// Y=128, Co=128 (0 after -128), Cg=128 (0 after -128)
	// Should give R=128, G=128, B=128
	luma := []byte{128}
	co := []byte{128} // 128-128 = 0
	cg := []byte{128} // 128-128 = 0
	alpha := []byte{255}

	result, err := aycoCgToRGBA(luma, co, cg, alpha, 1, 1, 1, 1)
	require.NoError(t, err)

	// t = Y - Cg = 128 - 0 = 128
	// R = t + Co = 128 + 0 = 128
	// G = Y + Cg = 128 + 0 = 128
	// B = t - Co = 128 - 0 = 128
	assert.Equal(t, []byte{128, 128, 128, 255}, result)
}

func TestAycoCgToRGBA_WithChroma(t *testing.T) {
	// Test with some chroma offset
	// Y=200, Co=160 (32 after -128), Cg=192 (64 after -128)
	luma := []byte{200}
	co := []byte{160} // 160-128 = 32
	cg := []byte{192} // 192-128 = 64
	alpha := []byte{255}

	result, err := aycoCgToRGBA(luma, co, cg, alpha, 1, 1, 1, 1)
	require.NoError(t, err)

	// t = Y - Cg = 200 - 64 = 136
	// R = t + Co = 136 + 32 = 168
	// G = Y + Cg = 200 + 64 = 255 (clamped from 264)
	// B = t - Co = 136 - 32 = 104
	assert.Equal(t, byte(168), result[0], "R")
	assert.Equal(t, byte(255), result[1], "G") // clamped
	assert.Equal(t, byte(104), result[2], "B")
	assert.Equal(t, byte(255), result[3], "A")
}

func TestParseBitmapStream_InvalidData(t *testing.T) {
	// Too short
	_, err := ParseBitmapStream([]byte{1, 2, 3})
	assert.Error(t, err)

	// Invalid color loss level (0)
	data := make([]byte, 20)
	data[16] = 0 // ColorLossLevel = 0 (invalid, must be 1-7)
	_, err = ParseBitmapStream(data)
	assert.Error(t, err)

	// Invalid color loss level (8)
	data[16] = 8 // ColorLossLevel = 8 (invalid)
	_, err = ParseBitmapStream(data)
	assert.Error(t, err)
}

func TestParseBitmapStream_ValidHeader(t *testing.T) {
	// Minimal valid stream with color loss level 1
	data := make([]byte, 20)
	// Luma plane byte count = 0 (all counts are 0 means no plane data)
	// This is technically invalid but tests header parsing
	data[16] = 1 // ColorLossLevel = 1 (valid)
	data[17] = 0 // ChromaSubsamplingLevel = 0 (no subsampling)

	// Set non-zero plane counts for a valid stream
	// For simplicity, set all to 1 byte each
	data[0] = 1  // LumaPlaneByteCount = 1
	data[4] = 1  // OrangeChromaPlaneByteCount = 1
	data[8] = 1  // GreenChromaPlaneByteCount = 1
	data[12] = 0 // AlphaPlaneByteCount = 0 (no alpha)

	// Add the plane data
	data = append(data, 128, 128, 128) // Luma, Co, Cg planes (1 byte each)

	stream, err := ParseBitmapStream(data)
	require.NoError(t, err)
	assert.Equal(t, uint8(1), stream.ColorLossLevel)
	assert.Equal(t, uint8(0), stream.ChromaSubsamplingLevel)
	assert.Equal(t, []byte{128}, stream.LumaPlane)
	assert.Equal(t, []byte{128}, stream.OrangeChromaPlane)
	assert.Equal(t, []byte{128}, stream.GreenChromaPlane)
	assert.Nil(t, stream.AlphaPlane)
}

func TestNSCodecGUID(t *testing.T) {
	// Verify the GUID matches the expected value from MS-RDPNSC
	// GUID: CA8D1BB9-000F-154F-589F-AE2D1A87E2D6
	expected := [16]byte{
		0xB9, 0x1B, 0x8D, 0xCA, 0x0F, 0x00, 0x4F, 0x15,
		0x58, 0x9F, 0xAE, 0x2D, 0x1A, 0x87, 0xE2, 0xD6,
	}
	assert.Equal(t, expected, NSCodecGUID)
}

func TestRLEDecompress_RawData(t *testing.T) {
	// If data length equals expected size, it's raw data
	raw := []byte{1, 2, 3, 4, 5}
	result, err := decompressPlane(raw, 5)
	require.NoError(t, err)
	assert.Equal(t, raw, result)
}

func TestRLEDecompress_TooLarge(t *testing.T) {
	// Data larger than expected is invalid
	data := []byte{1, 2, 3, 4, 5, 6}
	_, err := decompressPlane(data, 5)
	assert.Error(t, err)
}

// =============================================================================
// RLE Decompression Tests
// =============================================================================

func TestRLEDecompress_TooShort(t *testing.T) {
	// Data less than 4 bytes is invalid (needs EndData)
	_, err := rleDecompress([]byte{1, 2, 3}, 10)
	assert.ErrorIs(t, err, ErrDecompressionFail)
}

func TestRLEDecompress_RunSegment(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		expectedSize int
		wantErr      bool
	}{
		{
			name: "simple run segment",
			// Header 0x82 = 0x80 | 0x02 = run of length 2, value 0xAA
			// EndData: 0x01, 0x02, 0x03, 0x04
			data:         []byte{0x82, 0xAA, 0x01, 0x02, 0x03, 0x04},
			expectedSize: 6, // 2 bytes of run + 4 bytes EndData
			wantErr:      false,
		},
		{
			name: "extended run segment",
			// Header 0x80 = run with extended length
			// Extended length: 0x02 => actual length = 0x02 + 128 = 130
			// Value: 0x55
			// EndData: 0x01, 0x02, 0x03, 0x04
			data:         []byte{0x80, 0x02, 0x55, 0x01, 0x02, 0x03, 0x04},
			expectedSize: 134, // 130 bytes of run + 4 bytes EndData
			wantErr:      false,
		},
		{
			name: "literal segment",
			// Header 0x03 = literal of length 3
			// Literal data: 0xAA, 0xBB, 0xCC
			// EndData: 0x01, 0x02, 0x03, 0x04
			data:         []byte{0x03, 0xAA, 0xBB, 0xCC, 0x01, 0x02, 0x03, 0x04},
			expectedSize: 7, // 3 bytes literal + 4 bytes EndData
			wantErr:      false,
		},
		{
			name: "extended literal segment",
			// Header 0x00 = literal with extended length
			// Extended length: 0x05 => actual length = 0x05 + 128 = 133
			data:         append(append([]byte{0x00, 0x05}, make([]byte, 133)...), []byte{0x01, 0x02, 0x03, 0x04}...),
			expectedSize: 137, // 133 bytes literal + 4 bytes EndData
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rleDecompress(tt.data, tt.expectedSize)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedSize)
			}
		})
	}
}

func TestRLEDecompress_MixedSegments(t *testing.T) {
	// Mixed run and literal segments
	// Run: 0x83 = run length 3, value 0xAA => outputs: AA AA AA
	// Literal: 0x02 = literal length 2, data: 0xBB, 0xCC => outputs: BB CC
	// EndData: 0x01, 0x02, 0x03, 0x04
	data := []byte{0x83, 0xAA, 0x02, 0xBB, 0xCC, 0x01, 0x02, 0x03, 0x04}
	expectedSize := 9 // 3 run + 2 literal + 4 EndData

	result, err := rleDecompress(data, expectedSize)
	require.NoError(t, err)
	assert.Len(t, result, expectedSize)
	// First 3 bytes should be 0xAA (run)
	assert.Equal(t, []byte{0xAA, 0xAA, 0xAA}, result[0:3])
	// Next 2 bytes should be literal
	assert.Equal(t, []byte{0xBB, 0xCC}, result[3:5])
	// Last 4 bytes should be EndData
	assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, result[5:9])
}

func TestRLEDecompress_PaddingWhenSmall(t *testing.T) {
	// When decompressed data is smaller than expectedSize, it should pad with zeros
	// Simple run: 0x81 = run length 1, value 0xFF
	// EndData: 0x01, 0x02, 0x03, 0x04
	data := []byte{0x81, 0xFF, 0x01, 0x02, 0x03, 0x04}
	expectedSize := 10 // Larger than actual decompressed data

	result, err := rleDecompress(data, expectedSize)
	require.NoError(t, err)
	assert.Len(t, result, expectedSize)
}

func TestRLEDecompress_ExtendedRunValid(t *testing.T) {
	// Extended run: 0x80 means extended length follows
	// Length byte 0x01 => actual run length = 1 + 128 = 129
	// Value byte 0xAB
	// EndData: 0x01, 0x02, 0x03, 0x04
	data := []byte{0x80, 0x01, 0xAB, 0x01, 0x02, 0x03, 0x04}
	result, err := rleDecompress(data, 133) // 129 + 4 = 133
	require.NoError(t, err)
	assert.Len(t, result, 133)
}

func TestRLEDecompress_ExtendedLiteralMissingData(t *testing.T) {
	// Extended literal but not enough data
	data := []byte{0x00, 0x10, 0x01, 0x02, 0x03, 0x04} // 0x00, 0x10 means 144 bytes literal, but only 4 bytes follow
	_, err := rleDecompress(data, 200)
	assert.Error(t, err)
}

// =============================================================================
// NSCodec Decode Tests
// =============================================================================

func TestDecode_ValidStream(t *testing.T) {
	// Create a minimal valid NSCodec stream
	// 1x1 pixel with Y=128, Co=128, Cg=128 (gray pixel)
	data := make([]byte, 20)
	binary.LittleEndian.PutUint32(data[0:4], 1)  // LumaPlaneByteCount
	binary.LittleEndian.PutUint32(data[4:8], 1)  // OrangeChromaPlaneByteCount
	binary.LittleEndian.PutUint32(data[8:12], 1) // GreenChromaPlaneByteCount
	binary.LittleEndian.PutUint32(data[12:16], 0) // AlphaPlaneByteCount (no alpha)
	data[16] = 1 // ColorLossLevel
	data[17] = 0 // ChromaSubsamplingLevel (no subsampling)
	// Reserved bytes 18, 19

	// Add plane data
	data = append(data, 128, 128, 128) // Y, Co, Cg

	result, err := Decode(data, 1, 1)
	require.NoError(t, err)
	assert.Len(t, result, 4) // 1 pixel * 4 bytes (RGBA)

	// Expected: R=128, G=128, B=128, A=255 (gray pixel, no alpha plane)
	assert.Equal(t, byte(128), result[0], "R")
	assert.Equal(t, byte(128), result[1], "G")
	assert.Equal(t, byte(128), result[2], "B")
	assert.Equal(t, byte(255), result[3], "A")
}

func TestDecode_WithAlphaPlane(t *testing.T) {
	data := make([]byte, 20)
	binary.LittleEndian.PutUint32(data[0:4], 1)   // LumaPlaneByteCount
	binary.LittleEndian.PutUint32(data[4:8], 1)   // OrangeChromaPlaneByteCount
	binary.LittleEndian.PutUint32(data[8:12], 1)  // GreenChromaPlaneByteCount
	binary.LittleEndian.PutUint32(data[12:16], 1) // AlphaPlaneByteCount
	data[16] = 1 // ColorLossLevel
	data[17] = 0 // ChromaSubsamplingLevel

	// Add plane data including alpha
	data = append(data, 128, 128, 128, 200) // Y, Co, Cg, Alpha

	result, err := Decode(data, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, byte(200), result[3], "Alpha should be 200")
}

func TestDecode_WithChromaSubsampling(t *testing.T) {
	// 2x2 pixel image with chroma subsampling
	// Luma plane: 8x2 (roundUpToMultiple(2, 8) = 8, height = 2)
	// Chroma plane: 4x1 (8/2, roundUpToMultiple(2,2)/2 = 1)
	data := make([]byte, 20)
	binary.LittleEndian.PutUint32(data[0:4], 16)  // LumaPlaneByteCount (8*2)
	binary.LittleEndian.PutUint32(data[4:8], 4)   // OrangeChromaPlaneByteCount (4*1)
	binary.LittleEndian.PutUint32(data[8:12], 4)  // GreenChromaPlaneByteCount (4*1)
	binary.LittleEndian.PutUint32(data[12:16], 0) // AlphaPlaneByteCount
	data[16] = 1 // ColorLossLevel
	data[17] = 1 // ChromaSubsamplingLevel (enabled)

	// Add luma plane data (16 bytes)
	lumaData := make([]byte, 16)
	for i := range lumaData {
		lumaData[i] = 128
	}
	data = append(data, lumaData...)

	// Add chroma plane data (4 bytes each)
	chromaData := make([]byte, 4)
	for i := range chromaData {
		chromaData[i] = 128
	}
	data = append(data, chromaData...) // Co
	data = append(data, chromaData...) // Cg

	result, err := Decode(data, 2, 2)
	require.NoError(t, err)
	assert.Len(t, result, 16) // 2x2 pixels * 4 bytes
}

func TestDecode_WithColorLoss(t *testing.T) {
	data := make([]byte, 20)
	binary.LittleEndian.PutUint32(data[0:4], 1)  // LumaPlaneByteCount
	binary.LittleEndian.PutUint32(data[4:8], 1)  // OrangeChromaPlaneByteCount
	binary.LittleEndian.PutUint32(data[8:12], 1) // GreenChromaPlaneByteCount
	binary.LittleEndian.PutUint32(data[12:16], 0) // AlphaPlaneByteCount
	data[16] = 3 // ColorLossLevel (shift by 2)
	data[17] = 0 // ChromaSubsamplingLevel

	data = append(data, 128, 64, 64) // Y, Co, Cg

	result, err := Decode(data, 1, 1)
	require.NoError(t, err)
	assert.Len(t, result, 4)
}

func TestDecode_InvalidStream(t *testing.T) {
	// Too short
	_, err := Decode([]byte{1, 2, 3}, 1, 1)
	assert.Error(t, err)
}

func TestDecode_PlaneSizeMismatch(t *testing.T) {
	// Plane sizes don't match actual data
	data := make([]byte, 20)
	binary.LittleEndian.PutUint32(data[0:4], 100) // LumaPlaneByteCount (too large)
	binary.LittleEndian.PutUint32(data[4:8], 1)
	binary.LittleEndian.PutUint32(data[8:12], 1)
	binary.LittleEndian.PutUint32(data[12:16], 0)
	data[16] = 1
	data[17] = 0

	_, err := Decode(data, 1, 1)
	assert.Error(t, err)
}

func TestParseBitmapStream_OrangeChromaPlaneSizeMismatch(t *testing.T) {
	data := make([]byte, 20)
	binary.LittleEndian.PutUint32(data[0:4], 1)   // LumaPlaneByteCount
	binary.LittleEndian.PutUint32(data[4:8], 100) // OrangeChromaPlaneByteCount (too large)
	binary.LittleEndian.PutUint32(data[8:12], 1)
	binary.LittleEndian.PutUint32(data[12:16], 0)
	data[16] = 1
	data[17] = 0
	data = append(data, 128) // Only 1 byte for luma

	_, err := ParseBitmapStream(data)
	assert.ErrorIs(t, err, ErrInvalidPlaneSize)
}

func TestParseBitmapStream_GreenChromaPlaneSizeMismatch(t *testing.T) {
	data := make([]byte, 20)
	binary.LittleEndian.PutUint32(data[0:4], 1)
	binary.LittleEndian.PutUint32(data[4:8], 1)
	binary.LittleEndian.PutUint32(data[8:12], 100) // GreenChromaPlaneByteCount (too large)
	binary.LittleEndian.PutUint32(data[12:16], 0)
	data[16] = 1
	data[17] = 0
	data = append(data, 128, 128) // Luma and Orange

	_, err := ParseBitmapStream(data)
	assert.ErrorIs(t, err, ErrInvalidPlaneSize)
}

func TestParseBitmapStream_AlphaPlaneSizeMismatch(t *testing.T) {
	data := make([]byte, 20)
	binary.LittleEndian.PutUint32(data[0:4], 1)
	binary.LittleEndian.PutUint32(data[4:8], 1)
	binary.LittleEndian.PutUint32(data[8:12], 1)
	binary.LittleEndian.PutUint32(data[12:16], 100) // AlphaPlaneByteCount (too large)
	data[16] = 1
	data[17] = 0
	data = append(data, 128, 128, 128) // Luma, Orange, Green

	_, err := ParseBitmapStream(data)
	assert.ErrorIs(t, err, ErrInvalidPlaneSize)
}

func TestBitmapStreamDecode_DecompressPlaneError(t *testing.T) {
	// Create stream that will fail during plane decompression
	stream := &BitmapStream{
		LumaPlaneByteCount:         10,  // Mismatch with plane size
		OrangeChromaPlaneByteCount: 1,
		GreenChromaPlaneByteCount:  1,
		AlphaPlaneByteCount:        0,
		ColorLossLevel:             1,
		ChromaSubsamplingLevel:     0,
		LumaPlane:                  []byte{1, 2, 3}, // Too short, will fail decompression
		OrangeChromaPlane:          []byte{128},
		GreenChromaPlane:           []byte{128},
		AlphaPlane:                 nil,
	}

	_, err := stream.Decode(1, 1)
	assert.Error(t, err)
}

func TestDecompressPlane_RLECompressed(t *testing.T) {
	// Test with RLE compressed data (smaller than expected size)
	// Run segment: 0x85 = run length 5, value 0xAA
	// EndData: 0x01, 0x02, 0x03, 0x04
	compressed := []byte{0x85, 0xAA, 0x01, 0x02, 0x03, 0x04}
	result, err := decompressPlane(compressed, 9)
	require.NoError(t, err)
	assert.Len(t, result, 9)
}

// =============================================================================
// Chroma Super Sample Edge Cases
// =============================================================================

func TestChromaSuperSample_SinglePixel(t *testing.T) {
	src := []byte{100}
	result := chromaSuperSample(src, 1, 1, 2, 2)
	expected := []byte{100, 100, 100, 100}
	assert.Equal(t, expected, result)
}

func TestChromaSuperSample_OutOfBounds(t *testing.T) {
	// Test edge case where source index would be out of bounds
	src := []byte{1, 2}
	result := chromaSuperSample(src, 2, 1, 4, 2)
	assert.Len(t, result, 8)
}

func TestChromaSuperSample_SmallSource(t *testing.T) {
	// Test with small non-empty source
	src := []byte{100, 200}
	result := chromaSuperSample(src, 2, 1, 4, 2)
	assert.Len(t, result, 8)
	// First row: 100, 100, 200, 200
	// Second row: 100, 100, 200, 200 (srcY clamped to srcHeight-1)
	assert.Equal(t, byte(100), result[0])
	assert.Equal(t, byte(100), result[1])
	assert.Equal(t, byte(200), result[2])
	assert.Equal(t, byte(200), result[3])
}

// =============================================================================
// AYCoCg to RGBA Edge Cases
// =============================================================================

func TestAycoCgToRGBA_EmptyPlanes(t *testing.T) {
	result, err := aycoCgToRGBA([]byte{}, []byte{}, []byte{}, nil, 0, 0, 0, 0)
	require.NoError(t, err)
	assert.Len(t, result, 0)
}

func TestAycoCgToRGBA_NilAlpha(t *testing.T) {
	luma := []byte{128}
	co := []byte{128}
	cg := []byte{128}

	result, err := aycoCgToRGBA(luma, co, cg, nil, 1, 1, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, byte(255), result[3], "Alpha should default to 255")
}

func TestAycoCgToRGBA_PlaneIndexOutOfBounds(t *testing.T) {
	// Larger image dimensions than plane data
	luma := []byte{128}
	co := []byte{128}
	cg := []byte{128}
	alpha := []byte{200}

	result, err := aycoCgToRGBA(luma, co, cg, alpha, 1, 1, 2, 2)
	require.NoError(t, err)
	assert.Len(t, result, 16) // 2x2*4
}

func TestAycoCgToRGBA_ClampingBehavior(t *testing.T) {
	tests := []struct {
		name       string
		y, co, cg  byte
		expectedR  byte
		expectedG  byte
		expectedB  byte
	}{
		{
			name:      "max values clamp high",
			y:         255,
			co:        255, // co = 255 - 128 = 127
			cg:        255, // cg = 255 - 128 = 127
			expectedR: 255, // clamped
			expectedG: 255, // clamped
			expectedB: 1,   // 255 - 127 - 127 = 1
		},
		{
			name:      "min values clamp low",
			y:         0,
			co:        0,   // co = 0 - 128 = -128
			cg:        0,   // cg = 0 - 128 = -128
			expectedR: 0,   // clamped from negative
			expectedG: 0,   // clamped from negative
			expectedB: 255, // clamped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			luma := []byte{tt.y}
			co := []byte{tt.co}
			cg := []byte{tt.cg}

			result, err := aycoCgToRGBA(luma, co, cg, nil, 1, 1, 1, 1)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedR, result[0], "R")
			assert.Equal(t, tt.expectedG, result[1], "G")
			assert.Equal(t, tt.expectedB, result[2], "B")
		})
	}
}

// =============================================================================
// Restore Color Loss Edge Cases
// =============================================================================

func TestRestoreColorLoss_AllLevels(t *testing.T) {
	tests := []struct {
		level    uint8
		input    byte
		expected byte
	}{
		{1, 100, 100}, // No change
		{2, 50, 100},  // shift by 1
		{3, 25, 100},  // shift by 2
		{4, 12, 96},   // shift by 3
		{5, 6, 96},    // shift by 4
		{6, 3, 96},    // shift by 5
		{7, 1, 64},    // shift by 6
	}

	for _, tt := range tests {
		t.Run("level_"+string(rune('0'+tt.level)), func(t *testing.T) {
			result := restoreColorLoss([]byte{tt.input}, tt.level)
			assert.Equal(t, tt.expected, result[0])
		})
	}
}

func TestRestoreColorLoss_EmptyInput(t *testing.T) {
	result := restoreColorLoss([]byte{}, 3)
	assert.Len(t, result, 0)
}

// =============================================================================
// UTF-16 Encoding Tests
// =============================================================================

func TestEncode_EmptyString(t *testing.T) {
	result := Encode("")
	assert.Len(t, result, 0)
}

func TestEncode_ASCIIString(t *testing.T) {
	result := Encode("ABC")
	// UTF-16LE: A=0x41,0x00 B=0x42,0x00 C=0x43,0x00
	expected := []byte{0x41, 0x00, 0x42, 0x00, 0x43, 0x00}
	assert.Equal(t, expected, result)
}

func TestEncode_UnicodeString(t *testing.T) {
	result := Encode("日本")
	// 日 = U+65E5, 本 = U+672C in UTF-16LE
	assert.Len(t, result, 4) // 2 characters * 2 bytes each
}

func TestEncode_MixedString(t *testing.T) {
	result := Encode("A日")
	// A = 0x41,0x00, 日 = 0xE5,0x65 in LE
	assert.Len(t, result, 4)
	assert.Equal(t, byte(0x41), result[0])
	assert.Equal(t, byte(0x00), result[1])
}

func TestEncode_SurrogatePairs(t *testing.T) {
	// Emoji requires surrogate pairs in UTF-16
	result := Encode("😀")
	assert.Len(t, result, 4) // Surrogate pair = 4 bytes
}
// =============================================================================
// RLE Decompression Tests
// =============================================================================

func TestRLEDecompress16(t *testing.T) {
	// Test the 16-bit RLE decompressor
	// Simple test: color run of 5 white pixels
	// Order: REGULAR_COLOR_RUN (0x3) with run length 5, followed by pixel 0xFFFF
	srcBuffer := []byte{0x65, 0xFF, 0xFF} // 0x65 = (0x3 << 5) | 5, pixel=0xFFFF
	destBuffer := make([]byte, 10)        // 5 pixels * 2 bytes

	result := RLEDecompress16(srcBuffer, destBuffer, 10)
	assert.True(t, result, "RLEDecompress16 should succeed")
}

func TestRLEDecompress16_EmptyBuffers(t *testing.T) {
	result := RLEDecompress16([]byte{}, []byte{}, 0)
	assert.True(t, result, "Empty buffer should return true (no data to process)")
}

func TestRLEDecompress24(t *testing.T) {
	// Test the 24-bit RLE decompressor
	srcBuffer := []byte{0x65, 0xFF, 0xFF, 0xFF} // color run of 5 white pixels
	destBuffer := make([]byte, 15)              // 5 pixels * 3 bytes

	result := RLEDecompress24(srcBuffer, destBuffer, 15)
	assert.True(t, result, "RLEDecompress24 should succeed")
}

func TestRLEDecompress32(t *testing.T) {
	// Test the 32-bit RLE decompressor
	srcBuffer := []byte{0x65, 0xFF, 0xFF, 0xFF} // color run of 5 white pixels (3 bytes per pixel in src)
	destBuffer := make([]byte, 20)              // 5 pixels * 4 bytes

	result := RLEDecompress32(srcBuffer, destBuffer, 20)
	assert.True(t, result, "RLEDecompress32 should succeed")
}

// =============================================================================
// RLE Code Constants Tests
// =============================================================================

func TestCodeConstants(t *testing.T) {
	// Verify code constants are correctly defined
	assert.Equal(t, 0x0, RegularBgRun)
	assert.Equal(t, 0x1, RegularFgRun)
	assert.Equal(t, 0x2, RegularFgBgImage)
	assert.Equal(t, 0x3, RegularColorRun)
	assert.Equal(t, 0x4, RegularColorImage)

	assert.Equal(t, 0xC, LiteSetFgFgRun)
	assert.Equal(t, 0xD, LiteSetFgFgBgImage)
	assert.Equal(t, 0xE, LiteDitheredRun)

	assert.Equal(t, 0xF0, MegaMegaBgRun)
	assert.Equal(t, 0xF1, MegaMegaFgRun)
	assert.Equal(t, 0xF2, MegaMegaFgBgImage)
	assert.Equal(t, 0xF3, MegaMegaColorRun)
	assert.Equal(t, 0xF4, MegaMegaColorImage)
	assert.Equal(t, 0xF6, MegaMegaSetFgRun)
	assert.Equal(t, 0xF7, MegaMegaSetFgBgImage)
	assert.Equal(t, 0xF8, MegaMegaDitheredRun)

	assert.Equal(t, 0xF9, SpecialFgBg1)
	assert.Equal(t, 0xFA, SpecialFgBg2)
	assert.Equal(t, 0xFD, White)
	assert.Equal(t, 0xFE, Black)
}
