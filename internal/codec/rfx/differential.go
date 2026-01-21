package rfx

// DifferentialDecode decodes differentially-encoded coefficients.
// In RemoteFX, the LL3 (DC) subband is differentially encoded -
// each coefficient is stored as the difference from the previous value.
// This function reverses that encoding by computing running sums.
//
// Must be called AFTER RLGR decoding on the LL3 subband (buffer[4032:4096]).
func DifferentialDecode(buffer []int16, size int) {
	if len(buffer) < size {
		return
	}
	for i := 1; i < size; i++ {
		buffer[i] += buffer[i-1]
	}
}
