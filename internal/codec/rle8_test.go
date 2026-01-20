package codec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteFgBgImage8_FirstLine(t *testing.T) {
	// Test foreground/background writing on first line
	dest := make([]byte, 10)
	destIdx := 0
	rowDelta := 10

	// Bitmask 0xAA = 10101010
	// FgBgBitmasks = [0x01, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x80]
	// So bits 1,3,5,7 are set (positions 1,2,4,6 in 0-indexed)
	bitmask := byte(0xAA)
	fgPel := byte(0xFF)
	cBits := 8
	firstLine := true

	newIdx := WriteFgBgImage8(dest, destIdx, rowDelta, bitmask, fgPel, cBits, firstLine)

	require.Equal(t, 8, newIdx)
	// Position 0: bitmask & 0x01 = 0, so 0
	// Position 1: bitmask & 0x02 = 1, so fgPel
	// etc.
	require.Equal(t, byte(0x00), dest[0])
	require.Equal(t, byte(0xFF), dest[1])
	require.Equal(t, byte(0x00), dest[2])
	require.Equal(t, byte(0xFF), dest[3])
	require.Equal(t, byte(0x00), dest[4])
	require.Equal(t, byte(0xFF), dest[5])
	require.Equal(t, byte(0x00), dest[6])
	require.Equal(t, byte(0xFF), dest[7])
}

func TestWriteFgBgImage8_NotFirstLine(t *testing.T) {
	// Test foreground/background XOR with previous line
	dest := make([]byte, 20)
	rowDelta := 10

	// Set up previous line
	for i := 0; i < 10; i++ {
		dest[i] = 0x55 // Previous line values
	}

	destIdx := 10 // Start at second line
	bitmask := byte(0x0F) // Lower 4 bits set
	fgPel := byte(0xFF)
	cBits := 4
	firstLine := false

	newIdx := WriteFgBgImage8(dest, destIdx, rowDelta, bitmask, fgPel, cBits, firstLine)

	require.Equal(t, 14, newIdx)
	// Position 0: bitmask & 0x01 = 1, so XOR(0x55, 0xFF) = 0xAA
	require.Equal(t, byte(0xAA), dest[10])
	// Position 1: bitmask & 0x02 = 1, so XOR(0x55, 0xFF) = 0xAA
	require.Equal(t, byte(0xAA), dest[11])
}

func TestWriteFgBgImage8_DestBoundsCheck(t *testing.T) {
	// Test that we don't overflow dest buffer
	dest := make([]byte, 3)
	destIdx := 0
	bitmask := byte(0xFF)
	fgPel := byte(0xFF)
	cBits := 8 // Wants to write 8 bytes
	firstLine := true

	// Should stop at dest boundary
	newIdx := WriteFgBgImage8(dest, destIdx, 10, bitmask, fgPel, cBits, firstLine)
	require.Equal(t, 3, newIdx) // Only wrote 3 bytes
}

func TestRLEDecompress8_WhiteBlack(t *testing.T) {
	// Test white (0xFD) and black (0xFE) codes
	src := []byte{
		0xFD, // White
		0xFE, // Black
		0xFD, // White
	}
	dest := make([]byte, 3)

	result := RLEDecompress8(src, dest, 10)
	require.True(t, result)
	require.Equal(t, byte(0xFF), dest[0]) // White
	require.Equal(t, byte(0x00), dest[1]) // Black
	require.Equal(t, byte(0xFF), dest[2]) // White
}

func TestRLEDecompress8_ColorRun(t *testing.T) {
	// Test regular color run (0x6X where X is run length in lower 5 bits)
	// 0x64 = 0x60 (color run) | 0x04 (length 4)
	src := []byte{
		0x64, // Regular color run, length 4
		0xAB, // Color value
	}
	dest := make([]byte, 10)

	result := RLEDecompress8(src, dest, 10)
	require.True(t, result)
	// Should have 4 pixels of 0xAB
	for i := 0; i < 4; i++ {
		require.Equal(t, byte(0xAB), dest[i], "pixel %d", i)
	}
}

func TestRLEDecompress8_ColorImage(t *testing.T) {
	// Test regular color image (0x8X where X is run length in lower 5 bits)
	// 0x84 = 0x80 (color image) | 0x04 (length 4)
	// Color image copies raw bytes
	src := []byte{
		0x84,                   // Regular color image, length 4
		0x11, 0x22, 0x33, 0x44, // Raw pixels
	}
	dest := make([]byte, 10)

	result := RLEDecompress8(src, dest, 10)
	require.True(t, result)
	require.Equal(t, byte(0x11), dest[0])
	require.Equal(t, byte(0x22), dest[1])
	require.Equal(t, byte(0x33), dest[2])
	require.Equal(t, byte(0x44), dest[3])
}

func TestRLEDecompress8_EmptyInput(t *testing.T) {
	src := []byte{}
	dest := make([]byte, 10)

	result := RLEDecompress8(src, dest, 10)
	require.True(t, result)
}
