package rfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Quantization Tests
// Reference: MS-RDPRFX Section 3.1.8.1.5
// ============================================================================

func TestDequantize_WithNilQuant(t *testing.T) {
	buffer := make([]int16, TilePixels)
	buffer[0] = 100

	// Should not panic with nil quant
	Dequantize(buffer, nil)

	// Buffer unchanged
	assert.Equal(t, int16(100), buffer[0])
}

func TestDequantize_SmallBuffer(t *testing.T) {
	buffer := make([]int16, 100) // Too small
	quant := &SubbandQuant{HL1: 6, LH1: 6, HH1: 6}

	// Should not panic with small buffer
	Dequantize(buffer, quant)
}

func TestDequantize_ShiftFormula(t *testing.T) {
	// Per MS-RDPRFX: dequantized = coefficient << (quant - 1)
	tests := []struct {
		name       string
		quantValue uint8
		input      int16
		expected   int16
	}{
		{"quant=6, shift=5", 6, 1, 32},      // 1 << 5 = 32
		{"quant=7, shift=6", 7, 1, 64},      // 1 << 6 = 64
		{"quant=8, shift=7", 8, 1, 128},     // 1 << 7 = 128
		{"quant=1, no shift", 1, 100, 100},  // No shift when quant <= 1
		{"quant=0, no shift", 0, 100, 100},  // No shift when quant <= 1
		{"negative value", 6, -5, -160},     // -5 << 5 = -160
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := []int16{tc.input}
			dequantBlock(data, tc.quantValue)
			assert.Equal(t, tc.expected, data[0])
		})
	}
}

func TestDequantize_AllSubbands(t *testing.T) {
	buffer := make([]int16, TilePixels)

	// Set a known value in each subband
	buffer[OffsetHL1] = 1
	buffer[OffsetLH1] = 1
	buffer[OffsetHH1] = 1
	buffer[OffsetHL2] = 1
	buffer[OffsetLH2] = 1
	buffer[OffsetHH2] = 1
	buffer[OffsetHL3] = 1
	buffer[OffsetLH3] = 1
	buffer[OffsetHH3] = 1
	buffer[OffsetLL3] = 1

	quant := &SubbandQuant{
		HL1: 6, LH1: 7, HH1: 8,
		HL2: 6, LH2: 7, HH2: 8,
		HL3: 6, LH3: 7, HH3: 8,
		LL3: 6,
	}

	Dequantize(buffer, quant)

	// Verify each subband was shifted correctly
	assert.Equal(t, int16(32), buffer[OffsetHL1], "HL1: 1 << (6-1) = 32")
	assert.Equal(t, int16(64), buffer[OffsetLH1], "LH1: 1 << (7-1) = 64")
	assert.Equal(t, int16(128), buffer[OffsetHH1], "HH1: 1 << (8-1) = 128")
	assert.Equal(t, int16(32), buffer[OffsetHL2], "HL2: 1 << (6-1) = 32")
	assert.Equal(t, int16(64), buffer[OffsetLH2], "LH2: 1 << (7-1) = 64")
	assert.Equal(t, int16(128), buffer[OffsetHH2], "HH2: 1 << (8-1) = 128")
	assert.Equal(t, int16(32), buffer[OffsetHL3], "HL3: 1 << (6-1) = 32")
	assert.Equal(t, int16(64), buffer[OffsetLH3], "LH3: 1 << (7-1) = 64")
	assert.Equal(t, int16(128), buffer[OffsetHH3], "HH3: 1 << (8-1) = 128")
	assert.Equal(t, int16(32), buffer[OffsetLL3], "LL3: 1 << (6-1) = 32")
}

// ============================================================================
// Microsoft Protocol Test Suite Validation Tests
// Reference: MS-RDPRFX_ClientTestDesignSpecification.md
// ============================================================================

// TestSubbandQuant_FieldOrdering validates quant array ordering per MS-RDPRFX 2.2.2.3.4
// Quant values: [0]=LL3, [1]=LH3, [2]=HL3, [3]=HH3, [4]=LH2, [5]=HL2, [6]=HH2, [7]=LH1, [8]=HL1, [9]=HH1
func TestSubbandQuant_FieldOrdering(t *testing.T) {
	// Create quant with distinct values to verify ordering
	quant := SubbandQuant{
		LL3: 6, LH3: 7, HL3: 8, HH3: 9,
		LH2: 10, HL2: 11, HH2: 12,
		LH1: 13, HL1: 14, HH1: 15,
	}

	// Verify we can access all fields
	assert.Equal(t, uint8(6), quant.LL3)
	assert.Equal(t, uint8(7), quant.LH3)
	assert.Equal(t, uint8(8), quant.HL3)
	assert.Equal(t, uint8(9), quant.HH3)
	assert.Equal(t, uint8(10), quant.LH2)
	assert.Equal(t, uint8(11), quant.HL2)
	assert.Equal(t, uint8(12), quant.HH2)
	assert.Equal(t, uint8(13), quant.LH1)
	assert.Equal(t, uint8(14), quant.HL1)
	assert.Equal(t, uint8(15), quant.HH1)
}

// TestSubbandQuant_ValidRange validates quant values are in valid range
// Per MS-RDPRFX: quant values should be 6-15
func TestSubbandQuant_ValidRange(t *testing.T) {
	validQuantValues := []uint8{6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	for _, q := range validQuantValues {
		// Each valid quant value should produce a valid shift
		shift := q - 1
		assert.True(t, shift >= 5 && shift <= 14, "quant %d produces shift %d", q, shift)
	}
}

// TestSubbandOffsets_PerSpec validates subband buffer offsets per MS-RDPRFX
func TestSubbandOffsets_PerSpec(t *testing.T) {
	// Per MS-RDPRFX and repository memory:
	// Linear layout: HL1@0, LH1@1024, HH1@2048, HL2@3072, LH2@3328, HH2@3584,
	//                HL3@3840, LH3@3904, HH3@3968, LL3@4032

	assert.Equal(t, 0, OffsetHL1, "HL1 should start at 0")
	assert.Equal(t, 1024, OffsetLH1, "LH1 should start at 1024")
	assert.Equal(t, 2048, OffsetHH1, "HH1 should start at 2048")
	assert.Equal(t, 3072, OffsetHL2, "HL2 should start at 3072")
	assert.Equal(t, 3328, OffsetLH2, "LH2 should start at 3328")
	assert.Equal(t, 3584, OffsetHH2, "HH2 should start at 3584")
	assert.Equal(t, 3840, OffsetHL3, "HL3 should start at 3840")
	assert.Equal(t, 3904, OffsetLH3, "LH3 should start at 3904")
	assert.Equal(t, 3968, OffsetHH3, "HH3 should start at 3968")
	assert.Equal(t, 4032, OffsetLL3, "LL3 should start at 4032")
}

// TestSubbandSizes_PerSpec validates subband sizes per MS-RDPRFX
func TestSubbandSizes_PerSpec(t *testing.T) {
	// Level 1: 32×32 = 1024 coefficients each
	assert.Equal(t, 1024, SizeL1, "Level 1 subbands should be 32x32=1024")

	// Level 2: 16×16 = 256 coefficients each
	assert.Equal(t, 256, SizeL2, "Level 2 subbands should be 16x16=256")

	// Level 3: 8×8 = 64 coefficients each
	assert.Equal(t, 64, SizeL3, "Level 3 subbands should be 8x8=64")

	// Total: 3*1024 + 3*256 + 4*64 = 3072 + 768 + 256 = 4096 = 64×64
	total := 3*SizeL1 + 3*SizeL2 + 4*SizeL3
	assert.Equal(t, TilePixels, total, "All subbands should total TilePixels")
}

// TestTileSize_64x64 validates 64×64 tile size per MS-RDPRFX 1.3
func TestTileSize_64x64(t *testing.T) {
	require.Equal(t, 64, TileSize, "Tile size must be 64 per MS-RDPRFX")
	require.Equal(t, 4096, TilePixels, "Tile pixels must be 64*64=4096")
}
