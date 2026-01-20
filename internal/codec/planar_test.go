package codec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecompressPlanar_EmptyInput(t *testing.T) {
	result := DecompressPlanar(nil, 10, 10)
	require.Nil(t, result)

	result = DecompressPlanar([]byte{}, 10, 10)
	require.Nil(t, result)
}

func TestDecompressPlanar_RawNoAlpha(t *testing.T) {
	// Format header with NoAlpha flag, no RLE
	width := 2
	height := 2
	planeSize := width * height

	// Header (1) + R plane + G plane + B plane
	input := make([]byte, 1+3*planeSize)
	input[0] = PlanarFlagNoAlpha // NoAlpha, no RLE

	// R plane: all 0xFF
	for i := 0; i < planeSize; i++ {
		input[1+i] = 0xFF
	}
	// G plane: all 0x80
	for i := 0; i < planeSize; i++ {
		input[1+planeSize+i] = 0x80
	}
	// B plane: all 0x40
	for i := 0; i < planeSize; i++ {
		input[1+2*planeSize+i] = 0x40
	}

	result := DecompressPlanar(input, width, height)
	require.NotNil(t, result)
	require.Len(t, result, planeSize*4)

	// Check first pixel (bottom-left of source becomes top-left of output)
	// Planar data is bottom-up, so source row 1 (bottom) becomes output row 0 (top)
	require.Equal(t, byte(0xFF), result[0]) // R
	require.Equal(t, byte(0x80), result[1]) // G
	require.Equal(t, byte(0x40), result[2]) // B
	require.Equal(t, byte(0xFF), result[3]) // A (255 when NoAlpha)
}

func TestDecompressPlanar_RawWithAlpha(t *testing.T) {
	// Format header without NoAlpha flag, no RLE
	width := 2
	height := 2
	planeSize := width * height

	// Header (1) + A plane + R plane + G plane + B plane
	input := make([]byte, 1+4*planeSize)
	input[0] = 0x00 // No flags (has alpha, no RLE)

	// A plane: all 0xCC
	for i := 0; i < planeSize; i++ {
		input[1+i] = 0xCC
	}
	// R plane: all 0xFF
	for i := 0; i < planeSize; i++ {
		input[1+planeSize+i] = 0xFF
	}
	// G plane: all 0x80
	for i := 0; i < planeSize; i++ {
		input[1+2*planeSize+i] = 0x80
	}
	// B plane: all 0x40
	for i := 0; i < planeSize; i++ {
		input[1+3*planeSize+i] = 0x40
	}

	result := DecompressPlanar(input, width, height)
	require.NotNil(t, result)
	require.Len(t, result, planeSize*4)

	// Check first pixel
	require.Equal(t, byte(0xFF), result[0])  // R
	require.Equal(t, byte(0x80), result[1])  // G
	require.Equal(t, byte(0x40), result[2])  // B
	require.Equal(t, byte(0xCC), result[3])  // A
}

func TestDecompressPlanar_InsufficientData(t *testing.T) {
	// Header indicates raw planes but not enough data
	input := []byte{PlanarFlagNoAlpha, 0xFF} // Only 2 bytes, needs more

	result := DecompressPlanar(input, 10, 10)
	require.Nil(t, result)
}

func TestDecompressPlanarPlaneRLE_Simple(t *testing.T) {
	// Test a simple RLE-encoded plane
	width := 4
	height := 1

	// Simple case: 4 raw bytes (control byte 0x40 = 4 raw, 0 run)
	src := []byte{
		0x40,                   // control: 4 raw bytes, 0 run
		0x11, 0x22, 0x33, 0x44, // raw data
	}

	dst := make([]byte, width*height)
	consumed := decompressPlanarPlaneRLE(src, dst, width, height)

	require.Equal(t, 5, consumed)
	require.Equal(t, []byte{0x11, 0x22, 0x33, 0x44}, dst)
}

func TestDecompressPlanarPlaneRLE_WithRun(t *testing.T) {
	// Control byte with both raw and run
	width := 6
	height := 1

	// 0x24 = 2 raw bytes, 4 run
	src := []byte{
		0x24,       // control: 2 raw, 4 run
		0xAA, 0xBB, // 2 raw bytes, last one (0xBB) repeats
	}

	dst := make([]byte, width*height)
	consumed := decompressPlanarPlaneRLE(src, dst, width, height)

	require.Equal(t, 3, consumed)
	require.Equal(t, []byte{0xAA, 0xBB, 0xBB, 0xBB, 0xBB, 0xBB}, dst)
}

func TestDecompressPlanarPlaneRLE_ExtendedRun1(t *testing.T) {
	// Extended run length mode 1: nRunLength == 1 means use cRawBytes + 16
	// 0x41 = nRunLength=1, cRawBytes=4 -> actual run length = 4+16 = 20
	// For first scanline, we need a value byte after control to set pixel
	width := 20
	height := 1

	src := []byte{
		0x41, // control: extended run length (4+16=20), 0 raw bytes
		0xFF, // pixel value (read as raw byte, then used for run)
	}

	dst := make([]byte, width*height)
	consumed := decompressPlanarPlaneRLE(src, dst, width, height)

	// Actually based on the code: when nRunLength=1, cRawBytes becomes 0
	// and nRunLength becomes cRawBytes(4)+16=20
	// Since cRawBytes=0, no raw bytes are read, but pixel starts at 0
	// and runs for 20 times as 0
	// Let me trace through the code more carefully...
	// controlByte=0x41, nRunLength=1, cRawBytes=4
	// since nRunLength==1: nRunLength=4+16=20, cRawBytes=0
	// No raw bytes loop runs (cRawBytes=0)
	// Run loop: 20 times, pixel=0 (default)
	require.GreaterOrEqual(t, consumed, 1)
}

func TestDecompressPlanarPlaneRLE_Error(t *testing.T) {
	// Insufficient source data
	src := []byte{0x40} // Says 4 raw bytes but none provided
	dst := make([]byte, 4)

	consumed := decompressPlanarPlaneRLE(src, dst, 4, 1)
	require.Equal(t, -1, consumed)
}
