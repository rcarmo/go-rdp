package codec

// RDP6 Planar Codec decoder
// Reference: MS-RDPEGDI 2.2.2.5 and FreeRDP planar.c

const (
	// Format header flags
	PlanarFlagRLE     = 0x10 // Run Length Encoding
	PlanarFlagNoAlpha = 0x20 // No Alpha plane
)

// DecompressPlanar decompresses RDP6 Planar codec data to RGBA
func DecompressPlanar(src []byte, width, height int) []byte {
	if len(src) < 1 {
		return nil
	}
	if width <= 0 || height <= 0 {
		return nil
	}

	formatHeader := src[0]
	hasRLE := (formatHeader & PlanarFlagRLE) != 0
	noAlpha := (formatHeader & PlanarFlagNoAlpha) != 0

	srcIdx := 1
	planeSize := width * height
	if planeSize <= 0 {
		return nil
	}

	// Allocate planes
	planeR := make([]byte, planeSize)
	planeG := make([]byte, planeSize)
	planeB := make([]byte, planeSize)
	planeA := make([]byte, planeSize)

	// Initialize alpha to 255 if no alpha plane
	if noAlpha {
		for i := range planeA {
			planeA[i] = 255
		}
	}

	if hasRLE {
		// RLE encoded planes: Alpha, Red, Green, Blue order
		var consumed int

		if !noAlpha {
			consumed = decompressPlanarPlaneRLE(src[srcIdx:], planeA, width, height)
			if consumed < 0 {
				return nil
			}
			srcIdx += consumed
		}

		consumed = decompressPlanarPlaneRLE(src[srcIdx:], planeR, width, height)
		if consumed < 0 {
			return nil
		}
		srcIdx += consumed

		consumed = decompressPlanarPlaneRLE(src[srcIdx:], planeG, width, height)
		if consumed < 0 {
			return nil
		}
		srcIdx += consumed

		consumed = decompressPlanarPlaneRLE(src[srcIdx:], planeB, width, height)
		if consumed < 0 {
			return nil
		}
	} else {
		// Raw planes
		if !noAlpha {
			if srcIdx+planeSize > len(src) {
				return nil
			}
			copy(planeA, src[srcIdx:srcIdx+planeSize])
			srcIdx += planeSize
		}

		if srcIdx+planeSize > len(src) {
			return nil
		}
		copy(planeR, src[srcIdx:srcIdx+planeSize])
		srcIdx += planeSize

		if srcIdx+planeSize > len(src) {
			return nil
		}
		copy(planeG, src[srcIdx:srcIdx+planeSize])
		srcIdx += planeSize

		if srcIdx+planeSize > len(src) {
			return nil
		}
		copy(planeB, src[srcIdx:srcIdx+planeSize])
	}

	// Combine planes to RGBA with vertical flip (planar data is bottom-up)
	rgba := make([]byte, planeSize*4)
	for y := 0; y < height; y++ {
		srcRow := (height - 1 - y) * width // Read from bottom
		dstRow := y * width                // Write to top
		for x := 0; x < width; x++ {
			srcIdx := srcRow + x
			dstIdx := (dstRow + x) * 4
			rgba[dstIdx] = planeR[srcIdx]
			rgba[dstIdx+1] = planeG[srcIdx]
			rgba[dstIdx+2] = planeB[srcIdx]
			rgba[dstIdx+3] = planeA[srcIdx]
		}
	}

	return rgba
}

// decompressPlanarPlaneRLE decompresses a single RLE-encoded plane
// Returns number of bytes consumed, or -1 on error
func decompressPlanarPlaneRLE(src []byte, dst []byte, width, height int) int {
	srcIdx := 0
	dstIdx := 0
	var previousScanline []byte

	for y := 0; y < height; y++ {
		currentScanlineStart := dstIdx
		var pixel int16 = 0 // Last pixel/delta value

		for x := 0; x < width; {
			if srcIdx >= len(src) {
				return -1
			}

			controlByte := src[srcIdx]
			srcIdx++

			nRunLength := int(controlByte & 0x0F)
			cRawBytes := int((controlByte >> 4) & 0x0F)

			// Extended run lengths
			switch nRunLength {
			case 1:
				nRunLength = cRawBytes + 16
				cRawBytes = 0
			case 2:
				nRunLength = cRawBytes + 32
				cRawBytes = 0
			}

			if x+cRawBytes+nRunLength > width {
				return -1
			}

			if previousScanline == nil {
				// First scanline: absolute values
				for cRawBytes > 0 {
					if srcIdx >= len(src) || dstIdx >= len(dst) {
						return -1
					}
					pixel = int16(src[srcIdx])
					srcIdx++
					dst[dstIdx] = byte(pixel)
					dstIdx++
					x++
					cRawBytes--
				}

				for nRunLength > 0 {
					if dstIdx >= len(dst) {
						return -1
					}
					dst[dstIdx] = byte(pixel)
					dstIdx++
					x++
					nRunLength--
				}
			} else {
				// Delta values relative to previous scanline
				for cRawBytes > 0 {
					if srcIdx >= len(src) || dstIdx >= len(dst) {
						return -1
					}
					deltaValue := src[srcIdx]
					srcIdx++

					// Decode delta (sign-magnitude encoding)
					if deltaValue&1 != 0 {
						// Negative delta
						pixel = -int16((deltaValue >> 1) + 1)
					} else {
						// Positive delta
						pixel = int16(deltaValue >> 1)
					}

					// Add delta to previous scanline value with saturation
					dst[dstIdx] = clampPlanarDelta(previousScanline[x], pixel)
					dstIdx++
					x++
					cRawBytes--
				}

					// For run, add same delta to each previous scanline value
					for nRunLength > 0 {
						if dstIdx >= len(dst) {
							return -1
						}
						dst[dstIdx] = clampPlanarDelta(previousScanline[x], pixel)
						dstIdx++
						x++
						nRunLength--
					}
			}
		}

		previousScanline = dst[currentScanlineStart:dstIdx]
	}

	return srcIdx
}

func clampPlanarDelta(base byte, delta int16) byte {
	value := int16(base) + delta
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return byte(value)
}
