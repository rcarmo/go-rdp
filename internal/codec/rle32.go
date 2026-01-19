package codec

// ReadPixel32Src reads a 32-bit pixel from source buffer (3 bytes BGR, outputs with alpha)
func ReadPixel32Src(data []byte, idx int) uint32 {
	if idx+2 >= len(data) {
		return 0xFF000000
	}
	return uint32(data[idx]) | (uint32(data[idx+1]) << 8) | (uint32(data[idx+2]) << 16) | 0xFF000000
}

// ReadPixel32Dest reads a 32-bit pixel from destination buffer (4 bytes)
func ReadPixel32Dest(data []byte, idx int) uint32 {
	if idx+3 >= len(data) {
		return 0
	}
	return uint32(data[idx]) | (uint32(data[idx+1]) << 8) | (uint32(data[idx+2]) << 16) | (uint32(data[idx+3]) << 24)
}

// WritePixel32 writes a 32-bit pixel to the buffer
func WritePixel32(data []byte, idx int, pixel uint32) {
	if idx+3 >= len(data) {
		return
	}
	data[idx] = byte(pixel & 0xFF)
	data[idx+1] = byte((pixel >> 8) & 0xFF)
	data[idx+2] = byte((pixel >> 16) & 0xFF)
	data[idx+3] = byte((pixel >> 24) & 0xFF)
}

// WriteFgBgImage32 writes a foreground/background image for 32-bit color
func WriteFgBgImage32(dest []byte, destIdx int, rowDelta int, bitmask byte, fgPel uint32, cBits int, firstLine bool) int {
	for i := 0; i < cBits && i < 8; i++ {
		if destIdx+3 >= len(dest) {
			break
		}
		if firstLine {
			if bitmask&FgBgBitmasks[i] != 0 {
				WritePixel32(dest, destIdx, fgPel)
			} else {
				WritePixel32(dest, destIdx, 0xFF000000) // black with alpha
			}
		} else {
			xorPixel := ReadPixel32Dest(dest, destIdx-rowDelta)
			if bitmask&FgBgBitmasks[i] != 0 {
				WritePixel32(dest, destIdx, xorPixel^fgPel)
			} else {
				WritePixel32(dest, destIdx, xorPixel)
			}
		}
		destIdx += 4
	}
	return destIdx
}

// RLEDecompress32 decompresses 32-bit RLE compressed bitmap data.
// Per MS-RDPBCGR, 32-bit RLE stores pixels as 3 bytes (BGR) in compressed stream
// but outputs to 4 bytes per pixel (BGRX) buffer.
func RLEDecompress32(src []byte, dest []byte, rowDelta int) bool {
	srcIdx := 0
	destIdx := 0
	var fgPel uint32 = 0xFFFFFFFF // white with full alpha
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
					WritePixel32(dest, destIdx, fgPel)
					destIdx += 4
					runLength--
				}
				for runLength > 0 && destIdx < len(dest) {
					WritePixel32(dest, destIdx, 0xFF000000) // black with alpha
					destIdx += 4
					runLength--
				}
			} else {
				if fInsertFgPel {
					prevPel := ReadPixel32Dest(dest, destIdx-rowDelta)
					WritePixel32(dest, destIdx, prevPel^fgPel)
					destIdx += 4
					runLength--
				}
				for runLength > 0 && destIdx < len(dest) {
					prevPel := ReadPixel32Dest(dest, destIdx-rowDelta)
					WritePixel32(dest, destIdx, prevPel)
					destIdx += 4
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
				fgPel = ReadPixel32Src(src, srcIdx) // Read 3 bytes from source
				srcIdx += 3
			}

			for runLength > 0 && destIdx < len(dest) {
				if fFirstLine {
					WritePixel32(dest, destIdx, fgPel)
				} else {
					prevPel := ReadPixel32Dest(dest, destIdx-rowDelta)
					WritePixel32(dest, destIdx, prevPel^fgPel)
				}
				destIdx += 4
				runLength--
			}
			continue
		}

		// Dithered Run Orders
		if code == LiteDitheredRun || code == MegaMegaDitheredRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			pixelA := ReadPixel32Src(src, srcIdx) // Read 3 bytes
			srcIdx += 3
			pixelB := ReadPixel32Src(src, srcIdx) // Read 3 bytes
			srcIdx += 3

			for runLength > 0 && destIdx+8 <= len(dest) {
				WritePixel32(dest, destIdx, pixelA)
				destIdx += 4
				WritePixel32(dest, destIdx, pixelB)
				destIdx += 4
				runLength--
			}
			continue
		}

		// Color Run Orders
		if code == RegularColorRun || code == MegaMegaColorRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			pixel := ReadPixel32Src(src, srcIdx) // Read 3 bytes
			srcIdx += 3

			for runLength > 0 && destIdx < len(dest) {
				WritePixel32(dest, destIdx, pixel)
				destIdx += 4
				runLength--
			}
			continue
		}

		// Color Image Orders
		if code == RegularColorImage || code == MegaMegaColorImage {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			for runLength > 0 && destIdx < len(dest) && srcIdx+3 <= len(src) {
				pixel := ReadPixel32Src(src, srcIdx) // Read 3 bytes
				srcIdx += 3
				WritePixel32(dest, destIdx, pixel)
				destIdx += 4
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
				fgPel = ReadPixel32Src(src, srcIdx) // Read 3 bytes
				srcIdx += 3
			}

			for runLength > 0 && srcIdx < len(src) {
				bitmask := src[srcIdx]
				srcIdx++
				cBits := 8
				if runLength < 8 {
					cBits = runLength
				}
				destIdx = WriteFgBgImage32(dest, destIdx, rowDelta, bitmask, fgPel, cBits, fFirstLine)
				runLength -= cBits
			}
			continue
		}

		// Special Orders
		if code == SpecialFgBg1 {
			bitmask := byte(maskSpecialFgBg1)
			destIdx = WriteFgBgImage32(dest, destIdx, rowDelta, bitmask, fgPel, 8, fFirstLine)
			srcIdx++
			continue
		}

		if code == SpecialFgBg2 {
			bitmask := byte(maskSpecialFgBg2)
			destIdx = WriteFgBgImage32(dest, destIdx, rowDelta, bitmask, fgPel, 8, fFirstLine)
			srcIdx++
			continue
		}

		// White/Black Orders
		if code == White {
			WritePixel32(dest, destIdx, 0xFFFFFFFF)
			destIdx += 4
			srcIdx++
			continue
		}

		if code == Black {
			WritePixel32(dest, destIdx, 0xFF000000)
			destIdx += 4
			srcIdx++
			continue
		}

		// Unknown code, skip
		srcIdx++
	}

	return true
}
