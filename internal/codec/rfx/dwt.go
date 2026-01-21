package rfx

// Inverse 2D DWT (Discrete Wavelet Transform) using 5/3 LeGall wavelet.
//
// The coefficient buffer uses linear/packed layout:
// - HL1 (32×32) at offset 0
// - LH1 (32×32) at offset 1024
// - HH1 (32×32) at offset 2048
// - HL2 (16×16) at offset 3072
// - LH2 (16×16) at offset 3328
// - HH2 (16×16) at offset 3584
// - HL3 (8×8)   at offset 3840
// - LH3 (8×8)   at offset 3904
// - HH3 (8×8)   at offset 3968
// - LL3 (8×8)   at offset 4032
//
// This implementation follows FreeRDP's rfx_dwt.c algorithm.

// dwtTempBuffer is a pre-allocated buffer for DWT intermediate results.
// This avoids allocation in the hot path for WASM.
var dwtTempBuffer [TilePixels]int16

// InverseDWT2D performs 3-level inverse 2D DWT on tile coefficients.
// Input: 4096 int16 coefficients in packed subband order
// Output: 4096 int16 spatial-domain values (in-place, returns same slice)
func InverseDWT2D(buffer []int16) []int16 {
	if len(buffer) < TilePixels {
		return nil
	}

	// Use pre-allocated temp buffer
	temp := dwtTempBuffer[:]

	// Level 3: 8×8 → 16×16
	// Input: HL3(@3840), LH3(@3904), HH3(@3968), LL3(@4032)
	// Output: HL2-LH2-HH2 region starting at offset 3072
	idwt2DBlock(buffer, temp, 3840, 8)

	// Level 2: 16×16 → 32×32
	// Input: HL2(@3072), LH2(@3328), HH2(@3584), + L2 from level 3
	// Output: HL1-LH1-HH1 region starting at offset 0
	idwt2DBlock(buffer, temp, 3072, 16)

	// Level 1: 32×32 → 64×64
	// Input: HL1(@0), LH1(@1024), HH1(@2048), + L1 from level 2
	// Output: Full 64×64 spatial tile at offset 0
	idwt2DBlock(buffer, temp, 0, 32)

	return buffer
}

// idwt2DBlock performs one level of inverse 2D DWT.
// After this operation, the output occupies 4*size*size elements starting at offset.
// The subbands are stored as: HL, LH, HH, LL (each size*size elements).
//
// Algorithm from FreeRDP rfx_dwt.c (5/3 LeGall lifting):
// Horizontal pass combines columns: LL+HL and LH+HH
// Vertical pass combines rows to produce final output
func idwt2DBlock(buffer, temp []int16, offset, size int) {
	// Calculate subband positions
	// At input: HL at offset, LH at offset+size², HH at offset+2*size², LL at offset+3*size²
	size2 := size * size
	hlOfs := offset
	lhOfs := offset + size2
	hhOfs := offset + 2*size2
	llOfs := offset + 3*size2

	total := size * 2 // Output dimension

	// Horizontal pass: process each row
	// Output: L (left half) in temp[0:total*size], H (right half) in temp[total*size:]
	lDst := 0
	hDst := total * size
	for y := 0; y < size; y++ {
		// Combine LL and HL → L row
		idwt1DRow(buffer[llOfs+y*size:], buffer[hlOfs+y*size:], temp[lDst:], size)
		lDst += total
		// Combine LH and HH → H row
		idwt1DRow(buffer[lhOfs+y*size:], buffer[hhOfs+y*size:], temp[hDst:], size)
		hDst += total
	}

	// Vertical pass: process each column
	// Combine L and H columns to produce final output
	lSrc := 0
	hSrc := total * size
	for x := 0; x < total; x++ {
		idwt1DCol(temp, lSrc+x, hSrc+x, buffer, offset+x, total, size)
	}
}

// idwt1DRow performs horizontal 1D inverse DWT.
// low: low-frequency coefficients (LL or LH row)
// high: high-frequency coefficients (HL or HH row)
// dst: output buffer (length = 2*halfSize)
// halfSize: number of samples in each input
//
// 5/3 LeGall inverse lifting (from FreeRDP):
// even[n] = low[n] - ((high[n-1] + high[n] + 1) >> 1)
// odd[n] = (high[n] << 1) + ((even[n] + even[n+1]) >> 1)
func idwt1DRow(low, high []int16, dst []int16, halfSize int) {
	// First even sample (n=0): high[-1] mirrors to high[0]
	dst[0] = low[0] - ((high[0] + high[0] + 1) >> 1)

	// Even samples (n=1 to halfSize-1)
	for n := 1; n < halfSize; n++ {
		dst[n*2] = low[n] - ((high[n-1] + high[n] + 1) >> 1)
	}

	// Odd samples (n=0 to halfSize-2)
	for n := 0; n < halfSize-1; n++ {
		dst[n*2+1] = (high[n] << 1) + ((dst[n*2] + dst[n*2+2]) >> 1)
	}

	// Last odd sample (n=halfSize-1): even[n+1] mirrors to even[n]
	n := halfSize - 1
	dst[n*2+1] = (high[n] << 1) + ((dst[n*2] + dst[n*2]) >> 1)
}

// idwt1DCol performs vertical 1D inverse DWT on a column.
// src: source buffer containing L and H rows
// lOfs: offset to low-frequency column start
// hOfs: offset to high-frequency column start
// dst: destination buffer
// dstOfs: offset to output column start
// stride: distance between rows in source/destination
// halfSize: number of samples in each input column
func idwt1DCol(src []int16, lOfs, hOfs int, dst []int16, dstOfs, stride, halfSize int) {
	// First even sample (n=0): h[-1] mirrors to h[0]
	l0 := src[lOfs]
	h0 := src[hOfs]
	even0 := l0 - ((h0 + h0 + 1) >> 1)
	dst[dstOfs] = even0

	// Even samples (n=1 to halfSize-1)
	for n := 1; n < halfSize; n++ {
		ln := src[lOfs+n*stride]
		hPrev := src[hOfs+(n-1)*stride]
		hn := src[hOfs+n*stride]
		dst[dstOfs+n*2*stride] = ln - ((hPrev + hn + 1) >> 1)
	}

	// Odd samples (n=0 to halfSize-2)
	for n := 0; n < halfSize-1; n++ {
		hn := src[hOfs+n*stride]
		en := dst[dstOfs+n*2*stride]
		enNext := dst[dstOfs+(n*2+2)*stride]
		dst[dstOfs+(n*2+1)*stride] = (hn << 1) + ((en + enNext) >> 1)
	}

	// Last odd sample (n=halfSize-1): even[n+1] mirrors to even[n]
	n := halfSize - 1
	hn := src[hOfs+n*stride]
	en := dst[dstOfs+n*2*stride]
	dst[dstOfs+(n*2+1)*stride] = (hn << 1) + ((en + en) >> 1)
}
