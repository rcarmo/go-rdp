package codec

// RLE decompression for 8-bit color depth

// WriteFgBgImage8 writes a foreground/background image for 8-bit color
func WriteFgBgImage8(dest []byte, destIdx int, rowDelta int, bitmask byte, fgPel byte, cBits int, firstLine bool) int {
	for i := 0; i < cBits && i < 8; i++ {
		if destIdx >= len(dest) {
			break
		}
		if firstLine {
			if bitmask&FgBgBitmasks[i] != 0 {
				dest[destIdx] = fgPel
			} else {
				dest[destIdx] = 0
			}
		} else {
			xorPixel := dest[destIdx-rowDelta]
			if bitmask&FgBgBitmasks[i] != 0 {
				dest[destIdx] = xorPixel ^ fgPel
			} else {
				dest[destIdx] = xorPixel
			}
		}
		destIdx++
	}
	return destIdx
}

// RLEDecompress8 decompresses 8-bit RLE compressed bitmap data
func RLEDecompress8(src []byte, dest []byte, rowDelta int) bool {
	srcIdx := 0
	destIdx := 0
	var fgPel byte = 0xFF // white
	fInsertFgPel := false
	fFirstLine := true

	for srcIdx < len(src) && destIdx < len(dest) {
		// Check for end of first scanline
		if fFirstLine && destIdx >= rowDelta {
			fFirstLine = false
			fInsertFgPel = false
		}

		code := ExtractCodeID(src[srcIdx])

		// Background Run Orders
		if code == RegularBgRun || code == MegaMegaBgRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			if fFirstLine {
				if fInsertFgPel {
					dest[destIdx] = fgPel
					destIdx++
					runLength--
				}
				for runLength > 0 && destIdx < len(dest) {
					dest[destIdx] = 0
					destIdx++
					runLength--
				}
			} else {
				if fInsertFgPel {
					dest[destIdx] = dest[destIdx-rowDelta] ^ fgPel
					destIdx++
					runLength--
				}
				for runLength > 0 && destIdx < len(dest) {
					dest[destIdx] = dest[destIdx-rowDelta]
					destIdx++
					runLength--
				}
			}
			fInsertFgPel = true
			continue
		}

		fInsertFgPel = false

		// Foreground Run Orders
		if code == RegularFgRun || code == MegaMegaFgRun ||
			code == LiteSetFgFgRun || code == MegaMegaSetFgRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			if code == LiteSetFgFgRun || code == MegaMegaSetFgRun {
				if srcIdx < len(src) {
					fgPel = src[srcIdx]
					srcIdx++
				}
			}

			for runLength > 0 && destIdx < len(dest) {
				if fFirstLine {
					dest[destIdx] = fgPel
				} else {
					dest[destIdx] = dest[destIdx-rowDelta] ^ fgPel
				}
				destIdx++
				runLength--
			}
			continue
		}

		// Dithered Run Orders
		if code == LiteDitheredRun || code == MegaMegaDitheredRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			var pixelA, pixelB byte
			if srcIdx < len(src) {
				pixelA = src[srcIdx]
				srcIdx++
			}
			if srcIdx < len(src) {
				pixelB = src[srcIdx]
				srcIdx++
			}

			for runLength > 0 && destIdx+1 < len(dest) {
				dest[destIdx] = pixelA
				destIdx++
				dest[destIdx] = pixelB
				destIdx++
				runLength--
			}
			continue
		}

		// Color Run Orders
		if code == RegularColorRun || code == MegaMegaColorRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			var pixel byte
			if srcIdx < len(src) {
				pixel = src[srcIdx]
				srcIdx++
			}

			for runLength > 0 && destIdx < len(dest) {
				dest[destIdx] = pixel
				destIdx++
				runLength--
			}
			continue
		}

		// Color Image Orders
		if code == RegularColorImage || code == MegaMegaColorImage {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			for runLength > 0 && destIdx < len(dest) && srcIdx < len(src) {
				dest[destIdx] = src[srcIdx]
				srcIdx++
				destIdx++
				runLength--
			}
			continue
		}

		// Foreground/Background Image Orders
		if code == RegularFgBgImage || code == MegaMegaFgBgImage ||
			code == LiteSetFgFgBgImage || code == MegaMegaSetFgBgImage {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			if code == LiteSetFgFgBgImage || code == MegaMegaSetFgBgImage {
				if srcIdx < len(src) {
					fgPel = src[srcIdx]
					srcIdx++
				}
			}

			for runLength > 0 && srcIdx < len(src) {
				bitmask := src[srcIdx]
				srcIdx++
				cBits := 8
				if runLength < 8 {
					cBits = runLength
				}
				destIdx = WriteFgBgImage8(dest, destIdx, rowDelta, bitmask, fgPel, cBits, fFirstLine)
				runLength -= cBits
			}
			continue
		}

		// Special Orders
		if code == SpecialFgBg1 {
			bitmask := byte(maskSpecialFgBg1)
			destIdx = WriteFgBgImage8(dest, destIdx, rowDelta, bitmask, fgPel, 8, fFirstLine)
			srcIdx++
			continue
		}

		if code == SpecialFgBg2 {
			bitmask := byte(maskSpecialFgBg2)
			destIdx = WriteFgBgImage8(dest, destIdx, rowDelta, bitmask, fgPel, 8, fFirstLine)
			srcIdx++
			continue
		}

		// White/Black Orders
		if code == White {
			dest[destIdx] = 0xFF
			destIdx++
			srcIdx++
			continue
		}

		if code == Black {
			dest[destIdx] = 0x00
			destIdx++
			srcIdx++
			continue
		}

		// Unknown code, skip
		srcIdx++
	}

	return true
}
