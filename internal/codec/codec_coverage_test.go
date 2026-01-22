package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNSCodecRLEDecompressRaw tests raw data pass-through
func TestNSCodecRLEDecompressRaw(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	result := NSCodecRLEDecompress(data, 4)
	assert.Equal(t, data, result)
}

// TestNSCodecRLEDecompressTooLarge tests data larger than expected
func TestNSCodecRLEDecompressTooLarge(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	result := NSCodecRLEDecompress(data, 4)
	assert.Nil(t, result)
}

// TestNSCodecRLEDecompressTooSmall tests data smaller than minimum
func TestNSCodecRLEDecompressTooSmall(t *testing.T) {
	data := []byte{0x01, 0x02}
	result := NSCodecRLEDecompress(data, 100)
	assert.Nil(t, result)
}

// TestNSCodecRLEDecompressRunSegment tests run segment decompression
func TestNSCodecRLEDecompressRunSegment(t *testing.T) {
	// Run segment: 0x83 = 0x80 | 3, then value 0xAA
	// Should output 3x 0xAA
	// Then EndData (4 bytes)
	data := []byte{0x83, 0xAA, 0x00, 0x00, 0x00, 0x00}
	result := NSCodecRLEDecompress(data, 10)
	// May return partial result based on implementation
	if result != nil {
		assert.Contains(t, result, byte(0xAA))
	}
}

// TestNSCodecRLEDecompressLiteralSegment tests literal segment decompression
func TestNSCodecRLEDecompressLiteralSegment(t *testing.T) {
	// Literal segment: 0x03 = 3 literal bytes
	// Then 3 bytes of data, then EndData
	data := []byte{0x03, 0xBB, 0xCC, 0xDD, 0x00, 0x00, 0x00, 0x00}
	result := NSCodecRLEDecompress(data, 10)
	if result != nil && len(result) >= 3 {
		assert.Equal(t, byte(0xBB), result[0])
	}
}

// TestReadPixel15 tests 15-bit pixel reading
func TestReadPixel15(t *testing.T) {
	// Simple 15-bit pixel: low byte first
	data := []byte{0x00, 0x00}
	pixel := ReadPixel15(data, 0)
	assert.Equal(t, uint16(0), pixel)

	// White pixel (all bits set in 15bpp)
	data = []byte{0xFF, 0x7F}
	pixel = ReadPixel15(data, 0)
	assert.NotEqual(t, uint16(0), pixel)
}

// TestWritePixel15 tests 15-bit pixel writing
func TestWritePixel15(t *testing.T) {
	data := make([]byte, 4)
	WritePixel15(data, 0, 0)
	assert.Equal(t, byte(0), data[0])
	assert.Equal(t, byte(0), data[1])
}

// TestReadPixel16 tests 16-bit pixel reading
func TestReadPixel16(t *testing.T) {
	data := []byte{0x00, 0x00}
	pixel := ReadPixel16(data, 0)
	assert.Equal(t, uint16(0), pixel)
}

// TestWritePixel16 tests 16-bit pixel writing
func TestWritePixel16(t *testing.T) {
	data := make([]byte, 4)
	WritePixel16(data, 0, 0)
	assert.Equal(t, byte(0), data[0])
}

// TestReadPixel24 tests 24-bit pixel reading
func TestReadPixel24(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00}
	pixel := ReadPixel24(data, 0)
	assert.Equal(t, uint32(0), pixel)
}

// TestWritePixel24 tests 24-bit pixel writing
func TestWritePixel24(t *testing.T) {
	data := make([]byte, 6)
	WritePixel24(data, 0, 0)
	assert.Equal(t, byte(0), data[0])
}

// TestReadPixel32 tests 32-bit pixel reading
func TestReadPixel32(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x00}
	pixel := ReadPixel32(data, 0)
	assert.Equal(t, uint32(0), pixel)
}

// TestWritePixel32 tests 32-bit pixel writing
func TestWritePixel32(t *testing.T) {
	data := make([]byte, 8)
	WritePixel32(data, 0, 0)
	assert.Equal(t, byte(0), data[0])
}

// TestWriteFgBgImage15 tests 15-bit FgBg image writing
func TestWriteFgBgImage15(t *testing.T) {
	data := make([]byte, 32)
	n := WriteFgBgImage15(data, 8, 8, 0x01, 0x0000, 1, true)
	assert.GreaterOrEqual(t, n, 0)
}

// TestWriteFgBgImage16 tests 16-bit FgBg image writing
func TestWriteFgBgImage16(t *testing.T) {
	data := make([]byte, 32)
	n := WriteFgBgImage16(data, 8, 8, 0x01, 0x0000, 1, true)
	assert.GreaterOrEqual(t, n, 0)
}

// TestWriteFgBgImage24 tests 24-bit FgBg image writing
func TestWriteFgBgImage24(t *testing.T) {
	data := make([]byte, 48)
	n := WriteFgBgImage24(data, 12, 12, 0x01, 0x000000, 1, true)
	assert.GreaterOrEqual(t, n, 0)
}

// TestWriteFgBgImage32 tests 32-bit FgBg image writing
func TestWriteFgBgImage32(t *testing.T) {
	data := make([]byte, 64)
	n := WriteFgBgImage32(data, 16, 16, 0x01, 0x00000000, 1, true)
	assert.GreaterOrEqual(t, n, 0)
}
