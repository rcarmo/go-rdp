package rfx

// YCbCr to RGB conversion using ICT (Irreversible Color Transform).
// This is the inverse of the color transform used in JPEG 2000 and RemoteFX.
//
// MS-RDPRFX uses 11.5 fixed-point format:
// - Coefficients from DWT are already in 11.5 format (1 sign + 10 integer + 5 fractional)
// - Y requires a DC level shift of +4096 (which is 128 in 11.5 format: 128 << 5)
//
// ICT inverse transform (floating-point reference):
//
//	R = Y + 1.402525*Cr
//	G = Y - 0.343730*Cb - 0.714401*Cr
//	B = Y + 1.769905*Cb
//
// Fixed-point implementation follows FreeRDP prim_colors.c
// Using divisor=16 means we scale coefficients by 65536 and shift right by 16+5=21

// Fixed-point coefficients (scaled by 65536 with divisor=16)
const (
	// Level shift for Y: 128 in 11.5 format
	YLevelShift int32 = 4096 // 128 << 5

	// Fixed-point divisor (we use 16, so shift = 16+5 = 21)
	fpDivisor = 16
	fpShift   = fpDivisor + 5 // Total shift for final result

	// Coefficients scaled by 65536 (1 << 16)
	// R = Y + 1.402525*Cr
	CrToR int32 = 91916 // 1.402525 * 65536

	// G = Y - 0.343730*Cb - 0.714401*Cr
	CbToG int32 = 22527 // 0.343730 * 65536
	CrToG int32 = 46819 // 0.714401 * 65536

	// B = Y + 1.769905*Cb
	CbToB int32 = 115992 // 1.769905 * 65536

	// Rounding offset for fixed-point division
	fpRound int32 = 1 << (fpShift - 1) // 1 << 20 = 1048576
)

// YCbCrToRGBA converts YCbCr tile data to RGBA.
// y, cb, cr: int16 arrays of length TilePixels (4096) in 11.5 fixed-point format
// output: byte array of length TilePixels*4 (16384)
func YCbCrToRGBA(y, cb, cr []int16, output []byte) {
	if len(y) < TilePixels || len(cb) < TilePixels || len(cr) < TilePixels {
		return
	}
	if len(output) < TileRGBASize {
		return
	}

	for i := 0; i < TilePixels; i++ {
		// Get YCbCr values and scale Y by divisor (shift left by fpDivisor)
		// Y also needs DC level shift of +4096 (128 in 11.5 format)
		yVal := (int32(y[i]) + YLevelShift) << fpDivisor
		cbVal := int32(cb[i])
		crVal := int32(cr[i])

		// ICT inverse transform with fixed-point arithmetic
		r := fixedPointRound(yVal + CrToR*crVal)
		g := fixedPointRound(yVal - CbToG*cbVal - CrToG*crVal)
		b := fixedPointRound(yVal + CbToB*cbVal)

		// Clamp to [0, 255] and store as RGBA
		outIdx := i * 4
		output[outIdx+0] = clampToByte(r)
		output[outIdx+1] = clampToByte(g)
		output[outIdx+2] = clampToByte(b)
		output[outIdx+3] = 255 // Alpha = fully opaque
	}
}

// clampToByte clamps an int32 to [0, 255]
func clampToByte(v int32) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}

func fixedPointRound(value int32) int32 {
	if value >= 0 {
		return (value + fpRound) >> fpShift
	}
	return (value - fpRound) >> fpShift
}
