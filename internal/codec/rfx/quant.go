package rfx

// Subband layout for 64×64 tile coefficient buffer (linear/packed format):
// See rfx.go for offset constants (OffsetHL1, OffsetLH1, etc.)
//
// Dequantization: coefficient = coefficient << (quant - 1)
// Note: MS-RDPRFX specifies quant values 6-15, effective shift is (quant - 1)
//
// Quantization value ordering (indices 0-9 in quant array):
// [0]=LL3, [1]=LH3, [2]=HL3, [3]=HH3, [4]=LH2, [5]=HL2, [6]=HH2, [7]=LH1, [8]=HL1, [9]=HH1

// Dequantize applies inverse quantization to DWT coefficients.
// The buffer uses linear/packed layout matching FreeRDP.
func Dequantize(buffer []int16, quant *SubbandQuant) {
	if quant == nil || len(buffer) < TilePixels {
		return
	}

	// Level 1 subbands (32×32 = 1024 each)
	dequantBlock(buffer[OffsetHL1:OffsetHL1+SizeL1], quant.HL1)
	dequantBlock(buffer[OffsetLH1:OffsetLH1+SizeL1], quant.LH1)
	dequantBlock(buffer[OffsetHH1:OffsetHH1+SizeL1], quant.HH1)

	// Level 2 subbands (16×16 = 256 each)
	dequantBlock(buffer[OffsetHL2:OffsetHL2+SizeL2], quant.HL2)
	dequantBlock(buffer[OffsetLH2:OffsetLH2+SizeL2], quant.LH2)
	dequantBlock(buffer[OffsetHH2:OffsetHH2+SizeL2], quant.HH2)

	// Level 3 subbands (8×8 = 64 each)
	dequantBlock(buffer[OffsetHL3:OffsetHL3+SizeL3], quant.HL3)
	dequantBlock(buffer[OffsetLH3:OffsetLH3+SizeL3], quant.LH3)
	dequantBlock(buffer[OffsetHH3:OffsetHH3+SizeL3], quant.HH3)
	dequantBlock(buffer[OffsetLL3:OffsetLL3+SizeL3], quant.LL3)
}

// dequantBlock applies dequantization shift to a contiguous block.
// shift = quantValue - 1 per MS-RDPRFX spec
func dequantBlock(data []int16, quantValue uint8) {
	if quantValue <= 1 {
		return // No shift needed
	}
	shift := quantValue - 1
	for i := range data {
		data[i] <<= shift
	}
}
