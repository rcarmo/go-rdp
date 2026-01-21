package rfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: YCbCr values from the DWT are in 11.5 fixed-point format.
// The conversion expects this format. The Y level shift is +4096 (128<<5).

func TestYCbCrToRGBA_Gray(t *testing.T) {
	// Y=0 (in 11.5) + 4096 shift = 128 gray in output
	// After shift: (0 + 4096) << 16 = 268435456
	// RGB = 268435456 >> 21 = 128
	y := make([]int16, TilePixels)
	cb := make([]int16, TilePixels)
	cr := make([]int16, TilePixels)
	rgba := make([]byte, TileRGBASize)

	YCbCrToRGBA(y, cb, cr, rgba)

	// Check first pixel - should be gray (128, 128, 128)
	assert.Equal(t, byte(128), rgba[0], "R should be 128")
	assert.Equal(t, byte(128), rgba[1], "G should be 128")
	assert.Equal(t, byte(128), rgba[2], "B should be 128")
	assert.Equal(t, byte(255), rgba[3], "A should be 255")
}

func TestYCbCrToRGBA_White(t *testing.T) {
	// For white (255,255,255): Y in 11.5 = (255-128)<<5 = 127<<5 = 4064
	// After level shift: 4064+4096 = 8160, scaled << 16 then >> 21 = 255
	y := make([]int16, TilePixels)
	cb := make([]int16, TilePixels)
	cr := make([]int16, TilePixels)
	rgba := make([]byte, TileRGBASize)

	for i := range y {
		y[i] = 4064 // (255-128) << 5
	}

	YCbCrToRGBA(y, cb, cr, rgba)

	// Check first pixel - should be white (255, 255, 255)
	assert.InDelta(t, 255, int(rgba[0]), 2, "R should be ~255")
	assert.InDelta(t, 255, int(rgba[1]), 2, "G should be ~255")
	assert.InDelta(t, 255, int(rgba[2]), 2, "B should be ~255")
}

func TestYCbCrToRGBA_Black(t *testing.T) {
	// For black (0,0,0): Y in 11.5 = (0-128)<<5 = -128<<5 = -4096
	// After level shift: -4096+4096 = 0, RGB = 0
	y := make([]int16, TilePixels)
	cb := make([]int16, TilePixels)
	cr := make([]int16, TilePixels)
	rgba := make([]byte, TileRGBASize)

	for i := range y {
		y[i] = -4096 // (0-128) << 5
	}

	YCbCrToRGBA(y, cb, cr, rgba)

	// Check first pixel - should be black (0, 0, 0)
	assert.InDelta(t, 0, int(rgba[0]), 2, "R should be ~0")
	assert.InDelta(t, 0, int(rgba[1]), 2, "G should be ~0")
	assert.InDelta(t, 0, int(rgba[2]), 2, "B should be ~0")
}

func TestYCbCrToRGBA_Red(t *testing.T) {
	// Pure red (255,0,0) in YCbCr (BT.601): Y=76, Cb=-85, Cr=127
	// In 11.5: Y=(76-128)<<5=-1664, Cb=-85<<5=-2720, Cr=127<<5=4064
	y := make([]int16, TilePixels)
	cb := make([]int16, TilePixels)
	cr := make([]int16, TilePixels)
	rgba := make([]byte, TileRGBASize)

	for i := range y {
		y[i] = -1664  // (76-128) << 5
		cb[i] = -2720 // -85 << 5
		cr[i] = 4064  // 127 << 5
	}

	YCbCrToRGBA(y, cb, cr, rgba)

	// Should be close to red
	assert.InDelta(t, 255, int(rgba[0]), 20, "R should be ~255")
	assert.InDelta(t, 0, int(rgba[1]), 20, "G should be ~0")
	assert.InDelta(t, 0, int(rgba[2]), 20, "B should be ~0")
}

func TestYCbCrToRGBA_Green(t *testing.T) {
	// Pure green (0,255,0) in YCbCr (BT.601): Y=150, Cb=-74, Cr=-94
	// In 11.5: Y=(150-128)<<5=704, Cb=-74<<5=-2368, Cr=-94<<5=-3008
	y := make([]int16, TilePixels)
	cb := make([]int16, TilePixels)
	cr := make([]int16, TilePixels)
	rgba := make([]byte, TileRGBASize)

	for i := range y {
		y[i] = 704    // (150-128) << 5
		cb[i] = -2368 // -74 << 5
		cr[i] = -3008 // -94 << 5
	}

	YCbCrToRGBA(y, cb, cr, rgba)

	// Should be close to green
	assert.InDelta(t, 0, int(rgba[0]), 35, "R should be ~0")
	assert.InDelta(t, 255, int(rgba[1]), 35, "G should be ~255")
	assert.InDelta(t, 0, int(rgba[2]), 35, "B should be ~0")
}

func TestYCbCrToRGBA_Blue(t *testing.T) {
	// Pure blue (0,0,255) in YCbCr (BT.601): Y=29, Cb=127, Cr=-42
	// In 11.5: Y=(29-128)<<5=-3168, Cb=127<<5=4064, Cr=-42<<5=-1344
	y := make([]int16, TilePixels)
	cb := make([]int16, TilePixels)
	cr := make([]int16, TilePixels)
	rgba := make([]byte, TileRGBASize)

	for i := range y {
		y[i] = -3168 // (29-128) << 5
		cb[i] = 4064 // 127 << 5
		cr[i] = -1344 // -42 << 5
	}

	YCbCrToRGBA(y, cb, cr, rgba)

	// Should be close to blue
	assert.InDelta(t, 0, int(rgba[0]), 35, "R should be ~0")
	assert.InDelta(t, 0, int(rgba[1]), 35, "G should be ~0")
	assert.InDelta(t, 255, int(rgba[2]), 35, "B should be ~255")
}

func TestYCbCrToRGBA_SmallBuffers(t *testing.T) {
	// Test with buffers that are too small - should not panic
	y := make([]int16, 100)
	cb := make([]int16, 100)
	cr := make([]int16, 100)
	rgba := make([]byte, 100)

	// Should not panic, just return early
	YCbCrToRGBA(y, cb, cr, rgba)
}

func TestClampToByte(t *testing.T) {
	assert.Equal(t, byte(0), clampToByte(-100))
	assert.Equal(t, byte(0), clampToByte(-1))
	assert.Equal(t, byte(0), clampToByte(0))
	assert.Equal(t, byte(128), clampToByte(128))
	assert.Equal(t, byte(255), clampToByte(255))
	assert.Equal(t, byte(255), clampToByte(256))
	assert.Equal(t, byte(255), clampToByte(1000))
}

func TestDequantize(t *testing.T) {
	coeffs := make([]int16, TilePixels)

	// Set some test values in LL3 region (offset 4032, 64 elements)
	for i := 0; i < SizeL3; i++ {
		coeffs[OffsetLL3+i] = 10
	}

	quant := &SubbandQuant{
		LL3: 6, // shift = 5
		LH3: 6, HL3: 6, HH3: 6,
		LH2: 7, HL2: 7, HH2: 8,
		LH1: 8, HL1: 8, HH1: 9,
	}

	Dequantize(coeffs, quant)

	// LL3 values should be shifted left by 5 (quant-1)
	// 10 << 5 = 320
	for i := 0; i < SizeL3; i++ {
		assert.Equal(t, int16(320), coeffs[OffsetLL3+i])
	}
}

func TestDequantize_NilQuant(t *testing.T) {
	coeffs := make([]int16, TilePixels)
	coeffs[0] = 100

	// Should not panic with nil quant
	Dequantize(coeffs, nil)

	// Value should be unchanged
	assert.Equal(t, int16(100), coeffs[0])
}

func TestDequantize_LinearLayout(t *testing.T) {
	coeffs := make([]int16, TilePixels)

	// Set test values in different subbands
	coeffs[OffsetHL1] = 1    // HL1
	coeffs[OffsetLH1] = 2    // LH1
	coeffs[OffsetHH1] = 3    // HH1
	coeffs[OffsetHL2] = 4    // HL2
	coeffs[OffsetLH2] = 5    // LH2
	coeffs[OffsetHH2] = 6    // HH2
	coeffs[OffsetHL3] = 7    // HL3
	coeffs[OffsetLH3] = 8    // LH3
	coeffs[OffsetHH3] = 9    // HH3
	coeffs[OffsetLL3] = 10   // LL3

	quant := &SubbandQuant{
		LL3: 6, LH3: 6, HL3: 6, HH3: 6,  // shift 5
		LH2: 7, HL2: 7, HH2: 8,          // shift 6, 6, 7
		LH1: 8, HL1: 8, HH1: 9,          // shift 7, 7, 8
	}

	Dequantize(coeffs, quant)

	// Verify each subband was dequantized correctly
	assert.Equal(t, int16(1<<7), coeffs[OffsetHL1])  // HL1: 1 << 7 = 128
	assert.Equal(t, int16(2<<7), coeffs[OffsetLH1])  // LH1: 2 << 7 = 256
	assert.Equal(t, int16(3<<8), coeffs[OffsetHH1])  // HH1: 3 << 8 = 768
	assert.Equal(t, int16(4<<6), coeffs[OffsetHL2])  // HL2: 4 << 6 = 256
	assert.Equal(t, int16(5<<6), coeffs[OffsetLH2])  // LH2: 5 << 6 = 320
	assert.Equal(t, int16(6<<7), coeffs[OffsetHH2])  // HH2: 6 << 7 = 768
	assert.Equal(t, int16(7<<5), coeffs[OffsetHL3])  // HL3: 7 << 5 = 224
	assert.Equal(t, int16(8<<5), coeffs[OffsetLH3])  // LH3: 8 << 5 = 256
	assert.Equal(t, int16(9<<5), coeffs[OffsetHH3])  // HH3: 9 << 5 = 288
	assert.Equal(t, int16(10<<5), coeffs[OffsetLL3]) // LL3: 10 << 5 = 320
}

func TestParseQuantValues(t *testing.T) {
	data := []byte{
		0x65, // LL3=5, LH3=6
		0x87, // HL3=7, HH3=8
		0xA9, // LH2=9, HL2=10
		0xCB, // HH2=11, LH1=12
		0xED, // HL1=13, HH1=14
	}

	quant, err := ParseQuantValues(data)
	require.NoError(t, err)
	require.NotNil(t, quant)

	assert.Equal(t, uint8(5), quant.LL3)
	assert.Equal(t, uint8(6), quant.LH3)
	assert.Equal(t, uint8(7), quant.HL3)
	assert.Equal(t, uint8(8), quant.HH3)
	assert.Equal(t, uint8(9), quant.LH2)
	assert.Equal(t, uint8(10), quant.HL2)
	assert.Equal(t, uint8(11), quant.HH2)
	assert.Equal(t, uint8(12), quant.LH1)
	assert.Equal(t, uint8(13), quant.HL1)
	assert.Equal(t, uint8(14), quant.HH1)
}

func TestParseQuantValues_TooShort(t *testing.T) {
	data := []byte{0x65, 0x87, 0xA9} // Only 3 bytes, need 5

	quant, err := ParseQuantValues(data)
	assert.Error(t, err)
	assert.Nil(t, quant)
}

func TestDefaultQuant(t *testing.T) {
	quant := DefaultQuant()
	require.NotNil(t, quant)

	assert.Equal(t, uint8(6), quant.LL3)
	assert.Equal(t, uint8(6), quant.LH3)
	assert.Equal(t, uint8(6), quant.HL3)
	assert.Equal(t, uint8(6), quant.HH3)
	assert.Equal(t, uint8(7), quant.LH2)
	assert.Equal(t, uint8(7), quant.HL2)
	assert.Equal(t, uint8(8), quant.HH2)
	assert.Equal(t, uint8(8), quant.LH1)
	assert.Equal(t, uint8(8), quant.HL1)
	assert.Equal(t, uint8(9), quant.HH1)
}
