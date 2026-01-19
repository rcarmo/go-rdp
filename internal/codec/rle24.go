package codec

// ReadPixel24 reads a 24-bit pixel from the buffer
func ReadPixel24(data []byte, idx int) uint32 {
	if idx+2 >= len(data) {
		return 0
	}
	return uint32(data[idx]) | (uint32(data[idx+1]) << 8) | (uint32(data[idx+2]) << 16)
}

// WritePixel24 writes a 24-bit pixel to the buffer
func WritePixel24(data []byte, idx int, pixel uint32) {
	if idx+2 >= len(data) {
		return
	}
	data[idx] = byte(pixel & 0xFF)
	data[idx+1] = byte((pixel >> 8) & 0xFF)
	data[idx+2] = byte((pixel >> 16) & 0xFF)
}

// WriteFgBgImage24 writes a foreground/background image for 24-bit color
func WriteFgBgImage24(dest []byte, destIdx int, rowDelta int, bitmask byte, fgPel uint32, cBits int, firstLine bool) int {
	for i := 0; i < cBits && i < 8; i++ {
		if destIdx+2 >= len(dest) {
			break
		}
		if firstLine {
			if bitmask&FgBgBitmasks[i] != 0 {
				WritePixel24(dest, destIdx, fgPel)
			} else {
				WritePixel24(dest, destIdx, 0)
			}
		} else {
			xorPixel := ReadPixel24(dest, destIdx-rowDelta)
			if bitmask&FgBgBitmasks[i] != 0 {
				WritePixel24(dest, destIdx, xorPixel^fgPel)
			} else {
				WritePixel24(dest, destIdx, xorPixel)
			}
		}
		destIdx += 3
	}
	return destIdx
}

// RLEDecompress24 decompresses 24-bit RLE compressed bitmap data
func RLEDecompress24(src []byte, dest []byte, rowDelta int) bool {
	srcIdx := 0
	destIdx := 0
	var fgPel uint32 = 0xFFFFFF // white
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
					WritePixel24(dest, destIdx, fgPel)
					destIdx += 3
					runLength--
				}
				for runLength > 0 && destIdx < len(dest) {
					WritePixel24(dest, destIdx, 0)
					destIdx += 3
					runLength--
				}
			} else {
				if fInsertFgPel {
					prevPel := ReadPixel24(dest, destIdx-rowDelta)
					WritePixel24(dest, destIdx, prevPel^fgPel)
					destIdx += 3
					runLength--
				}
				for runLength > 0 && destIdx < len(dest) {
					prevPel := ReadPixel24(dest, destIdx-rowDelta)
					WritePixel24(dest, destIdx, prevPel)
					destIdx += 3
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
				fgPel = ReadPixel24(src, srcIdx)
				srcIdx += 3
			}

			for runLength > 0 && destIdx < len(dest) {
				if fFirstLine {
					WritePixel24(dest, destIdx, fgPel)
				} else {
					prevPel := ReadPixel24(dest, destIdx-rowDelta)
					WritePixel24(dest, destIdx, prevPel^fgPel)
				}
				destIdx += 3
				runLength--
			}
			continue
		}

		// Dithered Run Orders
		if code == LiteDitheredRun || code == MegaMegaDitheredRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			pixelA := ReadPixel24(src, srcIdx)
			srcIdx += 3
			pixelB := ReadPixel24(src, srcIdx)
			srcIdx += 3

			for runLength > 0 && destIdx+6 <= len(dest) {
				WritePixel24(dest, destIdx, pixelA)
				destIdx += 3
				WritePixel24(dest, destIdx, pixelB)
				destIdx += 3
				runLength--
			}
			continue
		}

		// Color Run Orders
		if code == RegularColorRun || code == MegaMegaColorRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			pixel := ReadPixel24(src, srcIdx)
			srcIdx += 3

			for runLength > 0 && destIdx < len(dest) {
				WritePixel24(dest, destIdx, pixel)
				destIdx += 3
				runLength--
			}
			continue
		}

		// Color Image Orders
		if code == RegularColorImage || code == MegaMegaColorImage {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			for runLength > 0 && destIdx < len(dest) && srcIdx+3 <= len(src) {
				pixel := ReadPixel24(src, srcIdx)
				srcIdx += 3
				WritePixel24(dest, destIdx, pixel)
				destIdx += 3
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
				fgPel = ReadPixel24(src, srcIdx)
				srcIdx += 3
			}

			for runLength > 0 && srcIdx < len(src) {
				bitmask := src[srcIdx]
				srcIdx++
				cBits := 8
				if runLength < 8 {
					cBits = runLength
				}
				destIdx = WriteFgBgImage24(dest, destIdx, rowDelta, bitmask, fgPel, cBits, fFirstLine)
				runLength -= cBits
			}
			continue
		}

		// Special Orders
		if code == SpecialFgBg1 {
			bitmask := byte(maskSpecialFgBg1)
			destIdx = WriteFgBgImage24(dest, destIdx, rowDelta, bitmask, fgPel, 8, fFirstLine)
			srcIdx++
			continue
		}

		if code == SpecialFgBg2 {
			bitmask := byte(maskSpecialFgBg2)
			destIdx = WriteFgBgImage24(dest, destIdx, rowDelta, bitmask, fgPel, 8, fFirstLine)
			srcIdx++
			continue
		}

		// White/Black Orders
		if code == White {
			WritePixel24(dest, destIdx, 0xFFFFFF)
			destIdx += 3
			srcIdx++
			continue
		}

		if code == Black {
			WritePixel24(dest, destIdx, 0x000000)
			destIdx += 3
			srcIdx++
			continue
		}

		// Unknown code, skip
		srcIdx++
	}

	return true
}
