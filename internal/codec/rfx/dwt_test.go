package rfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInverseDWT2D_NilInput(t *testing.T) {
	result := InverseDWT2D(nil)
	assert.Nil(t, result)
}

func TestInverseDWT2D_SmallInput(t *testing.T) {
	small := make([]int16, 100)
	result := InverseDWT2D(small)
	assert.Nil(t, result)
}

func TestInverseDWT2D_DCOnly(t *testing.T) {
	// Create coefficients with only DC component
	// In packed layout, LL3 is at offset 4032
	coeffs := make([]int16, TilePixels)
	coeffs[OffsetLL3] = 1000 // LL3[0,0] = 1000, all others = 0

	result := InverseDWT2D(coeffs)

	require.NotNil(t, result)
	require.Len(t, result, TilePixels)

	// With only DC component set, after inverse DWT the energy
	// should spread across the tile. All values should be similar.
	// Check that we have non-zero values
	hasNonZero := false
	for i := 0; i < TilePixels; i++ {
		if result[i] != 0 {
			hasNonZero = true
			break
		}
	}
	assert.True(t, hasNonZero, "expected non-zero values after inverse DWT")
}

func TestInverseDWT2D_AllZeros(t *testing.T) {
	coeffs := make([]int16, TilePixels)
	result := InverseDWT2D(coeffs)

	require.NotNil(t, result)
	require.Len(t, result, TilePixels)

	// All zeros in = all zeros out
	for i := 0; i < TilePixels; i++ {
		assert.Equal(t, int16(0), result[i])
	}
}

func TestIdwt1DRow_Size2(t *testing.T) {
	// Test with halfSize = 2 (4 elements output)
	low := []int16{100, 200}  // Low-frequency
	high := []int16{10, 20}   // High-frequency
	dst := make([]int16, 4)

	idwt1DRow(low, high, dst, 2)

	// Verify we get 4 reconstructed values
	require.Len(t, dst, 4)

	// Just verify no crash and values are reasonable
	for i := 0; i < 4; i++ {
		assert.True(t, dst[i] >= -1000 && dst[i] <= 1000, "value out of expected range at %d: %d", i, dst[i])
	}
}

func TestIdwt1DRow_Size4(t *testing.T) {
	// Test with halfSize = 4 (8 elements output)
	low := []int16{10, 20, 30, 40}
	high := []int16{1, 2, 3, 4}
	dst := make([]int16, 8)

	idwt1DRow(low, high, dst, 4)

	require.Len(t, dst, 8)

	// Verify all values are set
	for i := 0; i < 8; i++ {
		assert.True(t, dst[i] >= -100 && dst[i] <= 100, "value out of expected range at %d: %d", i, dst[i])
	}
}

func TestInverseDWT2D_UniformDC(t *testing.T) {
	// Fill LL3 with uniform value to check reconstruction spreads it
	coeffs := make([]int16, TilePixels)
	for i := 0; i < SizeL3; i++ {
		coeffs[OffsetLL3+i] = 128
	}

	result := InverseDWT2D(coeffs)
	require.NotNil(t, result)

	// Check that output has non-zero values
	sum := int32(0)
	for i := 0; i < TilePixels; i++ {
		sum += int32(result[i])
	}
	assert.NotEqual(t, int32(0), sum, "expected non-zero sum after inverse DWT")
}
