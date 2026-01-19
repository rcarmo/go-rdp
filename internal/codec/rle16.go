package codec

// ReadPixel16 reads a 16-bit pixel from the buffer
func ReadPixel16(data []byte, idx int) uint16 {
	if idx+1 >= len(data) {
		return 0
	}
	return uint16(data[idx]) | (uint16(data[idx+1]) << 8)
}

// WritePixel16 writes a 16-bit pixel to the buffer
func WritePixel16(data []byte, idx int, pixel uint16) {
	if idx+1 >= len(data) {
		return
	}
	data[idx] = byte(pixel & 0xFF)
	data[idx+1] = byte((pixel >> 8) & 0xFF)
}

// WriteFgBgImage16 writes a foreground/background image for 16-bit color
func WriteFgBgImage16(dest []byte, destIdx int, rowDelta int, bitmask byte, fgPel uint16, cBits int, firstLine bool) int {
	for i := 0; i < cBits && i < 8; i++ {
		if destIdx+1 >= len(dest) {
			break
		}
		if firstLine {
			if bitmask&FgBgBitmasks[i] != 0 {
				WritePixel16(dest, destIdx, fgPel)
			} else {
				WritePixel16(dest, destIdx, 0)
			}
		} else {
			xorPixel := ReadPixel16(dest, destIdx-rowDelta)
			if bitmask&FgBgBitmasks[i] != 0 {
				WritePixel16(dest, destIdx, xorPixel^fgPel)
			} else {
				WritePixel16(dest, destIdx, xorPixel)
			}
		}
		destIdx += 2
	}
	return destIdx
}

// RLEDecompress16 decompresses 16-bit RLE compressed bitmap data
func RLEDecompress16(src []byte, dest []byte, rowDelta int) bool {
	srcIdx := 0
	destIdx := 0
	var fgPel uint16 = 0xFFFF // white
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
					WritePixel16(dest, destIdx, fgPel)
					destIdx += 2
					runLength--
				}
				for runLength > 0 && destIdx < len(dest) {
					WritePixel16(dest, destIdx, 0)
					destIdx += 2
					runLength--
				}
			} else {
				if fInsertFgPel {
					prevPel := ReadPixel16(dest, destIdx-rowDelta)
					WritePixel16(dest, destIdx, prevPel^fgPel)
					destIdx += 2
					runLength--
				}
				for runLength > 0 && destIdx < len(dest) {
					prevPel := ReadPixel16(dest, destIdx-rowDelta)
					WritePixel16(dest, destIdx, prevPel)
					destIdx += 2
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
				fgPel = ReadPixel16(src, srcIdx)
				srcIdx += 2
			}

			for runLength > 0 && destIdx < len(dest) {
				if fFirstLine {
					WritePixel16(dest, destIdx, fgPel)
				} else {
					prevPel := ReadPixel16(dest, destIdx-rowDelta)
					WritePixel16(dest, destIdx, prevPel^fgPel)
				}
				destIdx += 2
				runLength--
			}
			continue
		}

		// Dithered Run Orders
		if code == LiteDitheredRun || code == MegaMegaDitheredRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			pixelA := ReadPixel16(src, srcIdx)
			srcIdx += 2
			pixelB := ReadPixel16(src, srcIdx)
			srcIdx += 2

			for runLength > 0 && destIdx+4 <= len(dest) {
				WritePixel16(dest, destIdx, pixelA)
				destIdx += 2
				WritePixel16(dest, destIdx, pixelB)
				destIdx += 2
				runLength--
			}
			continue
		}

		// Color Run Orders
		if code == RegularColorRun || code == MegaMegaColorRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			pixel := ReadPixel16(src, srcIdx)
			srcIdx += 2

			for runLength > 0 && destIdx < len(dest) {
				WritePixel16(dest, destIdx, pixel)
				destIdx += 2
				runLength--
			}
			continue
		}

		// Color Image Orders
		if code == RegularColorImage || code == MegaMegaColorImage {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			for runLength > 0 && destIdx < len(dest) && srcIdx+2 <= len(src) {
				pixel := ReadPixel16(src, srcIdx)
				srcIdx += 2
				WritePixel16(dest, destIdx, pixel)
				destIdx += 2
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
				fgPel = ReadPixel16(src, srcIdx)
				srcIdx += 2
			}

			for runLength > 0 && srcIdx < len(src) {
				bitmask := src[srcIdx]
				srcIdx++
				cBits := 8
				if runLength < 8 {
					cBits = runLength
				}
				destIdx = WriteFgBgImage16(dest, destIdx, rowDelta, bitmask, fgPel, cBits, fFirstLine)
				runLength -= cBits
			}
			continue
		}

		// Special Orders
		if code == SpecialFgBg1 {
			bitmask := byte(maskSpecialFgBg1)
			destIdx = WriteFgBgImage16(dest, destIdx, rowDelta, bitmask, fgPel, 8, fFirstLine)
			srcIdx++
			continue
		}

		if code == SpecialFgBg2 {
			bitmask := byte(maskSpecialFgBg2)
			destIdx = WriteFgBgImage16(dest, destIdx, rowDelta, bitmask, fgPel, 8, fFirstLine)
			srcIdx++
			continue
		}

		// White/Black Orders
		if code == White {
			WritePixel16(dest, destIdx, 0xFFFF)
			destIdx += 2
			srcIdx++
			continue
		}

		if code == Black {
			WritePixel16(dest, destIdx, 0x0000)
			destIdx += 2
			srcIdx++
			continue
		}

		// Unknown code, skip
		srcIdx++
	}

	return true
}
