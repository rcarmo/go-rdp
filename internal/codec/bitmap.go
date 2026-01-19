package codec

// FlipVertical flips bitmap data vertically (in-place).
// RDP sends bitmaps bottom-up, this flips them to top-down.
func FlipVertical(data []byte, width, height, bytesPerPixel int) {
	if height <= 1 {
		return
	}

	rowDelta := width * bytesPerPixel
	if rowDelta <= 0 || len(data) < height*rowDelta {
		return
	}

	tmp := make([]byte, rowDelta)
	half := height / 2

	for i := 0; i < half; i++ {
		topLine := i * rowDelta
		bottomLine := (height - 1 - i) * rowDelta

		copy(tmp, data[topLine:topLine+rowDelta])
		copy(data[topLine:topLine+rowDelta], data[bottomLine:bottomLine+rowDelta])
		copy(data[bottomLine:bottomLine+rowDelta], tmp)
	}
}

// RGB565ToRGBA converts 16-bit RGB565 to 32-bit RGBA
func RGB565ToRGBA(src []byte, dst []byte) {
	srcIdx := 0
	dstIdx := 0

	for srcIdx+1 < len(src) && dstIdx+3 < len(dst) {
		pel := uint16(src[srcIdx]) | (uint16(src[srcIdx+1]) << 8)

		r := (pel & 0xF800) >> 11
		g := (pel & 0x07E0) >> 5
		b := pel & 0x001F

		// Expand 5/6/5 to 8/8/8
		r = (r << 3) | (r >> 2)
		g = (g << 2) | (g >> 4)
		b = (b << 3) | (b >> 2)

		dst[dstIdx] = byte(r)
		dst[dstIdx+1] = byte(g)
		dst[dstIdx+2] = byte(b)
		dst[dstIdx+3] = 255

		srcIdx += 2
		dstIdx += 4
	}
}

// BGR24ToRGBA converts 24-bit BGR to 32-bit RGBA
func BGR24ToRGBA(src []byte, dst []byte) {
	srcIdx := 0
	dstIdx := 0

	for srcIdx+2 < len(src) && dstIdx+3 < len(dst) {
		dst[dstIdx] = src[srcIdx+2]   // R
		dst[dstIdx+1] = src[srcIdx+1] // G
		dst[dstIdx+2] = src[srcIdx]   // B
		dst[dstIdx+3] = 255

		srcIdx += 3
		dstIdx += 4
	}
}

// BGRA32ToRGBA converts 32-bit BGRA to 32-bit RGBA
func BGRA32ToRGBA(src []byte, dst []byte) {
	for i := 0; i+3 < len(src) && i+3 < len(dst); i += 4 {
		dst[i] = src[i+2]   // R
		dst[i+1] = src[i+1] // G
		dst[i+2] = src[i]   // B
		dst[i+3] = 255
	}
}

// ProcessBitmap handles decompression, flip, and color conversion in one call.
// Returns the RGBA output buffer on success, nil on failure.
func ProcessBitmap(src []byte, width, height, bpp int, isCompressed bool, rowDelta int) []byte {
	bytesPerPixel := bpp / 8
	rawSize := width * height * bytesPerPixel
	pixelCount := width * height

	var raw []byte

	if isCompressed {
		raw = make([]byte, rawSize)
		switch bpp {
		case 16:
			if !RLEDecompress16(src, raw, rowDelta) {
				return nil
			}
		case 24:
			if !RLEDecompress24(src, raw, rowDelta) {
				return nil
			}
		case 32:
			if !RLEDecompress32(src, raw, rowDelta) {
				return nil
			}
		default:
			return nil
		}
	} else {
		raw = src
	}

	// Flip vertically
	FlipVertical(raw, width, height, bytesPerPixel)

	// Convert to RGBA
	rgba := make([]byte, pixelCount*4)
	switch bpp {
	case 16:
		RGB565ToRGBA(raw, rgba)
	case 24:
		BGR24ToRGBA(raw, rgba)
	case 32:
		BGRA32ToRGBA(raw, rgba)
	default:
		return nil
	}

	return rgba
}
