package rfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBitStream_ReadBits(t *testing.T) {
	data := []byte{0xAB, 0xCD, 0xEF, 0x12}
	bs := NewBitStream(data)

	// Read 4 bits: should be 0xA (1010)
	assert.Equal(t, uint32(0x0A), bs.ReadBits(4))

	// Read 8 bits: should be 0xBC (remaining B from first byte + C from second)
	assert.Equal(t, uint32(0xBC), bs.ReadBits(8))

	// Read 4 bits: should be 0xD
	assert.Equal(t, uint32(0x0D), bs.ReadBits(4))
}

func TestBitStream_ReadBit(t *testing.T) {
	data := []byte{0x80} // 10000000
	bs := NewBitStream(data)

	// After refill, accumulator has data left-aligned in 32-bit
	// First bit should be 1
	bit1 := bs.ReadBit()
	assert.Equal(t, uint32(1), bit1, "first bit of 0x80 should be 1")

	bit2 := bs.ReadBit()
	assert.Equal(t, uint32(0), bit2, "second bit of 0x80 should be 0")

	bit3 := bs.ReadBit()
	assert.Equal(t, uint32(0), bit3, "third bit of 0x80 should be 0")
}

func TestBitStream_CountLeadingZeros(t *testing.T) {
	// 0x08 = 00001000, should have 4 leading zeros before the first 1
	data := []byte{0x08}
	bs := NewBitStream(data)

	count := bs.CountLeadingZeros()
	assert.Equal(t, 4, count, "0x08 should have 4 leading zeros")
}

func TestBitStream_CountLeadingOnes(t *testing.T) {
	// 0xF0 = 11110000, should have 4 leading ones before the first 0
	data := []byte{0xF0}
	bs := NewBitStream(data)

	count := bs.CountLeadingOnes()
	assert.Equal(t, 4, count, "0xF0 should have 4 leading ones")
}

func TestBitStream_RemainingBits(t *testing.T) {
	data := []byte{0xAB, 0xCD}
	bs := NewBitStream(data)

	assert.Equal(t, 16, bs.RemainingBits())

	bs.ReadBits(4)
	assert.Equal(t, 12, bs.RemainingBits())
}

func TestRLGRDecode_EmptyInput(t *testing.T) {
	output := make([]int16, TilePixels)
	err := RLGRDecode([]byte{}, RLGR1, output)

	require.NoError(t, err)
	// All zeros expected
	for i := 0; i < TilePixels; i++ {
		assert.Equal(t, int16(0), output[i])
	}
}

func TestRLGRDecode_BufferTooSmall(t *testing.T) {
	output := make([]int16, 100) // Too small
	err := RLGRDecode([]byte{0x00}, RLGR1, output)

	assert.Equal(t, ErrBufferTooSmall, err)
}

func TestRLGRDecode_RLGR1_BasicDecode(t *testing.T) {
	// This tests that the decoder doesn't crash on arbitrary data
	// and produces some output
	data := []byte{0x55, 0xAA, 0x55, 0xAA, 0xFF, 0x00, 0xFF, 0x00}
	output := make([]int16, TilePixels)

	err := RLGRDecode(data, RLGR1, output)
	require.NoError(t, err)

	// Just verify it completed without error
	// Actual values depend on the specific encoding
}

func TestRLGRDecode_RLGR3_BasicDecode(t *testing.T) {
	// Test RLGR3 mode (paired values)
	data := []byte{0x55, 0xAA, 0x55, 0xAA, 0xFF, 0x00, 0xFF, 0x00}
	output := make([]int16, TilePixels)

	err := RLGRDecode(data, RLGR3, output)
	require.NoError(t, err)
}
