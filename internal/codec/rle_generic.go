package codec

// PixelFormat defines the operations for a specific pixel bit depth in RLE decompression.
type PixelFormat[T uint8 | uint16 | uint32] struct {
	BytesPerPixel int
	WhitePixel    T
	BlackPixel    T
	ReadPixel     func(data []byte, idx int) T
	WritePixel    func(data []byte, idx int, pixel T)
}

// Pixel8 defines the 8-bit pixel format (1 byte per pixel).
var Pixel8 = PixelFormat[uint8]{
	BytesPerPixel: 1,
	WhitePixel:    0xFF,
	BlackPixel:    0x00,
	ReadPixel: func(data []byte, idx int) uint8 {
		if idx >= len(data) {
			return 0
		}
		return data[idx]
	},
	WritePixel: func(data []byte, idx int, pixel uint8) {
		if idx >= len(data) {
			return
		}
		data[idx] = pixel
	},
}

// Pixel15 defines the 15-bit pixel format (2 bytes per pixel, RGB555).
var Pixel15 = PixelFormat[uint16]{
	BytesPerPixel: 2,
	WhitePixel:    0x7FFF,
	BlackPixel:    0x0000,
	ReadPixel: func(data []byte, idx int) uint16 {
		if idx+1 >= len(data) {
			return 0
		}
		return uint16(data[idx]) | (uint16(data[idx+1]) << 8)
	},
	WritePixel: func(data []byte, idx int, pixel uint16) {
		if idx+1 >= len(data) {
			return
		}
		data[idx] = byte(pixel & 0xFF)
		data[idx+1] = byte((pixel >> 8) & 0xFF)
	},
}

// Pixel16 defines the 16-bit pixel format (2 bytes per pixel, RGB565).
var Pixel16 = PixelFormat[uint16]{
	BytesPerPixel: 2,
	WhitePixel:    0xFFFF,
	BlackPixel:    0x0000,
	ReadPixel: func(data []byte, idx int) uint16 {
		if idx+1 >= len(data) {
			return 0
		}
		return uint16(data[idx]) | (uint16(data[idx+1]) << 8)
	},
	WritePixel: func(data []byte, idx int, pixel uint16) {
		if idx+1 >= len(data) {
			return
		}
		data[idx] = byte(pixel & 0xFF)
		data[idx+1] = byte((pixel >> 8) & 0xFF)
	},
}

// Pixel24 defines the 24-bit pixel format (3 bytes per pixel, RGB888).
var Pixel24 = PixelFormat[uint32]{
	BytesPerPixel: 3,
	WhitePixel:    0xFFFFFF,
	BlackPixel:    0x000000,
	ReadPixel: func(data []byte, idx int) uint32 {
		if idx+2 >= len(data) {
			return 0
		}
		return uint32(data[idx]) | (uint32(data[idx+1]) << 8) | (uint32(data[idx+2]) << 16)
	},
	WritePixel: func(data []byte, idx int, pixel uint32) {
		if idx+2 >= len(data) {
			return
		}
		data[idx] = byte(pixel & 0xFF)
		data[idx+1] = byte((pixel >> 8) & 0xFF)
		data[idx+2] = byte((pixel >> 16) & 0xFF)
	},
}

// Pixel32 defines the 32-bit pixel format (4 bytes per pixel, RGBA8888).
var Pixel32 = PixelFormat[uint32]{
	BytesPerPixel: 4,
	WhitePixel:    0xFFFFFFFF,
	BlackPixel:    0x00000000,
	ReadPixel: func(data []byte, idx int) uint32 {
		if idx+3 >= len(data) {
			return 0
		}
		return uint32(data[idx]) | (uint32(data[idx+1]) << 8) |
			(uint32(data[idx+2]) << 16) | (uint32(data[idx+3]) << 24)
	},
	WritePixel: func(data []byte, idx int, pixel uint32) {
		if idx+3 >= len(data) {
			return
		}
		data[idx] = byte(pixel & 0xFF)
		data[idx+1] = byte((pixel >> 8) & 0xFF)
		data[idx+2] = byte((pixel >> 16) & 0xFF)
		data[idx+3] = byte((pixel >> 24) & 0xFF)
	},
}

// writeFgBgImage writes a foreground/background image using the given pixel format
func writeFgBgImage[T uint8 | uint16 | uint32](pf PixelFormat[T], dest []byte, destIdx int, rowDelta int, bitmask byte, fgPel T, cBits int, firstLine bool) int {
	bpp := pf.BytesPerPixel
	for i := 0; i < cBits && i < 8; i++ {
		if destIdx+bpp-1 >= len(dest) {
			break
		}
		if firstLine {
			if bitmask&FgBgBitmasks[i] != 0 {
				pf.WritePixel(dest, destIdx, fgPel)
			} else {
				pf.WritePixel(dest, destIdx, 0)
			}
		} else {
			xorPixel := pf.ReadPixel(dest, destIdx-rowDelta)
			if bitmask&FgBgBitmasks[i] != 0 {
				pf.WritePixel(dest, destIdx, xorPixel^fgPel)
			} else {
				pf.WritePixel(dest, destIdx, xorPixel)
			}
		}
		destIdx += bpp
	}
	return destIdx
}

// RLEDecompress decompresses RLE-encoded bitmap data using the specified pixel format.
func RLEDecompress[T uint8 | uint16 | uint32](pf PixelFormat[T], src []byte, dest []byte, rowDelta int) bool {
	srcIdx := 0
	destIdx := 0
	fgPel := pf.WhitePixel
	fInsertFgPel := false
	fFirstLine := true
	bpp := pf.BytesPerPixel

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
					pf.WritePixel(dest, destIdx, fgPel)
					destIdx += bpp
					runLength--
				}
				for runLength > 0 && destIdx < len(dest) {
					pf.WritePixel(dest, destIdx, 0)
					destIdx += bpp
					runLength--
				}
			} else {
				if fInsertFgPel {
					prevPel := pf.ReadPixel(dest, destIdx-rowDelta)
					pf.WritePixel(dest, destIdx, prevPel^fgPel)
					destIdx += bpp
					runLength--
				}
				for runLength > 0 && destIdx < len(dest) {
					prevPel := pf.ReadPixel(dest, destIdx-rowDelta)
					pf.WritePixel(dest, destIdx, prevPel)
					destIdx += bpp
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
				fgPel = pf.ReadPixel(src, srcIdx)
				srcIdx += bpp
			}

			for runLength > 0 && destIdx < len(dest) {
				if fFirstLine {
					pf.WritePixel(dest, destIdx, fgPel)
				} else {
					prevPel := pf.ReadPixel(dest, destIdx-rowDelta)
					pf.WritePixel(dest, destIdx, prevPel^fgPel)
				}
				destIdx += bpp
				runLength--
			}
			continue
		}

		// Dithered Run Orders
		if code == LiteDitheredRun || code == MegaMegaDitheredRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			pixelA := pf.ReadPixel(src, srcIdx)
			srcIdx += bpp
			pixelB := pf.ReadPixel(src, srcIdx)
			srcIdx += bpp

			for runLength > 0 && destIdx+2*bpp <= len(dest) {
				pf.WritePixel(dest, destIdx, pixelA)
				destIdx += bpp
				pf.WritePixel(dest, destIdx, pixelB)
				destIdx += bpp
				runLength--
			}
			continue
		}

		// Color Run Orders
		if code == RegularColorRun || code == MegaMegaColorRun {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			pixel := pf.ReadPixel(src, srcIdx)
			srcIdx += bpp

			for runLength > 0 && destIdx < len(dest) {
				pf.WritePixel(dest, destIdx, pixel)
				destIdx += bpp
				runLength--
			}
			continue
		}

		// Color Image Orders
		if code == RegularColorImage || code == MegaMegaColorImage {
			runLength, nextIdx := ExtractRunLength(code, src, srcIdx)
			srcIdx = nextIdx

			for runLength > 0 && destIdx < len(dest) && srcIdx+bpp <= len(src) {
				pixel := pf.ReadPixel(src, srcIdx)
				srcIdx += bpp
				pf.WritePixel(dest, destIdx, pixel)
				destIdx += bpp
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
				fgPel = pf.ReadPixel(src, srcIdx)
				srcIdx += bpp
			}

			for runLength > 0 && srcIdx < len(src) {
				bitmask := src[srcIdx]
				srcIdx++
				cBits := 8
				if runLength < 8 {
					cBits = runLength
				}
				destIdx = writeFgBgImage(pf, dest, destIdx, rowDelta, bitmask, fgPel, cBits, fFirstLine)
				runLength -= cBits
			}
			continue
		}

		// Special Orders
		if code == SpecialFgBg1 {
			bitmask := byte(maskSpecialFgBg1)
			destIdx = writeFgBgImage(pf, dest, destIdx, rowDelta, bitmask, fgPel, 8, fFirstLine)
			srcIdx++
			continue
		}

		if code == SpecialFgBg2 {
			bitmask := byte(maskSpecialFgBg2)
			destIdx = writeFgBgImage(pf, dest, destIdx, rowDelta, bitmask, fgPel, 8, fFirstLine)
			srcIdx++
			continue
		}

		// White/Black Orders
		if code == White {
			pf.WritePixel(dest, destIdx, pf.WhitePixel)
			destIdx += bpp
			srcIdx++
			continue
		}

		if code == Black {
			pf.WritePixel(dest, destIdx, pf.BlackPixel)
			destIdx += bpp
			srcIdx++
			continue
		}

		// Unknown code, skip
		srcIdx++
	}

	return true
}
