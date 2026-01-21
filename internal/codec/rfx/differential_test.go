package rfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDifferentialDecode_Basic(t *testing.T) {
	// Input: differences [1, 2, 3, 4, 5]
	// Output: cumulative sums [1, 3, 6, 10, 15]
	buffer := []int16{1, 2, 3, 4, 5}

	DifferentialDecode(buffer, 5)

	expected := []int16{1, 3, 6, 10, 15}
	assert.Equal(t, expected, buffer)
}

func TestDifferentialDecode_WithNegatives(t *testing.T) {
	// Input: [10, -2, 5, -3, 0]
	// Output: [10, 8, 13, 10, 10]
	buffer := []int16{10, -2, 5, -3, 0}

	DifferentialDecode(buffer, 5)

	expected := []int16{10, 8, 13, 10, 10}
	assert.Equal(t, expected, buffer)
}

func TestDifferentialDecode_SingleElement(t *testing.T) {
	buffer := []int16{42}

	DifferentialDecode(buffer, 1)

	// Single element should remain unchanged
	assert.Equal(t, int16(42), buffer[0])
}

func TestDifferentialDecode_Empty(t *testing.T) {
	// Should not panic
	buffer := []int16{}
	DifferentialDecode(buffer, 0)
}

func TestDifferentialDecode_PartialSize(t *testing.T) {
	// Only decode first 3 elements
	buffer := []int16{1, 2, 3, 100, 200}

	DifferentialDecode(buffer, 3)

	// First 3 should be decoded, rest unchanged
	assert.Equal(t, int16(1), buffer[0])  // 1
	assert.Equal(t, int16(3), buffer[1])  // 1+2
	assert.Equal(t, int16(6), buffer[2])  // 1+2+3
	assert.Equal(t, int16(100), buffer[3]) // unchanged
	assert.Equal(t, int16(200), buffer[4]) // unchanged
}
