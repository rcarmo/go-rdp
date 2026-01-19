package codec

// NSCodecRLEDecompress decompresses NSCodec RLE data for a single plane.
// This is different from bitmap RLE - NSCodec uses a simpler format with
// run segments and literal segments.
func NSCodecRLEDecompress(data []byte, expectedSize int) []byte {
	if len(data) == expectedSize {
		// Raw data, no decompression needed
		return data
	}

	if len(data) > expectedSize || len(data) < 4 {
		return nil
	}

	result := make([]byte, 0, expectedSize)
	offset := 0
	dataLen := len(data) - 4 // Exclude EndData

	for offset < dataLen && len(result) < expectedSize-4 {
		header := data[offset]
		offset++

		if header&0x80 != 0 {
			// Run segment: repeat single byte
			runLength := int(header & 0x7F)
			if runLength == 0 {
				if offset >= dataLen {
					return nil
				}
				runLength = int(data[offset]) + 128
				offset++
			}
			if offset >= dataLen {
				return nil
			}
			runValue := data[offset]
			offset++

			for i := 0; i < runLength && len(result) < expectedSize-4; i++ {
				result = append(result, runValue)
			}
		} else {
			// Literal segment: copy raw bytes
			literalLength := int(header)
			if literalLength == 0 {
				if offset >= dataLen {
					return nil
				}
				literalLength = int(data[offset]) + 128
				offset++
			}

			if offset+literalLength > dataLen {
				return nil
			}

			result = append(result, data[offset:offset+literalLength]...)
			offset += literalLength
		}
	}

	// Append EndData (last 4 bytes)
	if len(data) >= 4 {
		endData := data[len(data)-4:]
		for _, b := range endData {
			if len(result) < expectedSize {
				result = append(result, b)
			}
		}
	}

	// Pad with zeros if needed
	for len(result) < expectedSize {
		result = append(result, 0)
	}

	return result[:expectedSize]
}

// ChromaSuperSample upsamples chroma planes from subsampled to full resolution
func ChromaSuperSample(plane []byte, srcWidth, srcHeight, dstWidth, dstHeight int) []byte {
	result := make([]byte, dstWidth*dstHeight)

	for y := 0; y < dstHeight; y++ {
		srcY := y / 2
		if srcY >= srcHeight {
			srcY = srcHeight - 1
		}

		for x := 0; x < dstWidth; x++ {
			srcX := x / 2
			if srcX >= srcWidth {
				srcX = srcWidth - 1
			}

			srcIdx := srcY*srcWidth + srcX
			dstIdx := y*dstWidth + x

			if srcIdx < len(plane) {
				result[dstIdx] = plane[srcIdx]
			}
		}
	}

	return result
}

// RestoreColorLoss restores color values that were quantized during compression
func RestoreColorLoss(plane []byte, colorLossLevel uint8) []byte {
	if colorLossLevel <= 1 {
		return plane
	}

	shift := colorLossLevel - 1
	result := make([]byte, len(plane))

	for i, v := range plane {
		restored := int(v) << shift
		if restored > 255 {
			restored = 255
		}
		result[i] = byte(restored)
	}

	return result
}

// clampByteNS clamps a value to 0-255 range (used by NSCodec)
func clampByteNS(v int) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}

// AYCoCgToRGBA converts AYCoCg color space to RGBA
func AYCoCgToRGBA(luma, co, cg, alpha []byte, planeWidth, planeHeight, imgWidth, imgHeight int) []byte {
	rgba := make([]byte, imgWidth*imgHeight*4)

	for y := 0; y < imgHeight; y++ {
		for x := 0; x < imgWidth; x++ {
			planeIdx := y*planeWidth + x
			rgbaIdx := (y*imgWidth + x) * 4

			if planeIdx >= len(luma) || planeIdx >= len(co) || planeIdx >= len(cg) {
				continue
			}

			// Get YCoCg values (shifted to signed range)
			yVal := int(luma[planeIdx])
			coVal := int(co[planeIdx]) - 128
			cgVal := int(cg[planeIdx]) - 128

			// YCoCg to RGB conversion
			t := yVal - cgVal
			r := clampByteNS(t + coVal)
			g := clampByteNS(yVal + cgVal)
			b := clampByteNS(t - coVal)

			rgba[rgbaIdx+0] = r
			rgba[rgbaIdx+1] = g
			rgba[rgbaIdx+2] = b

			// Alpha
			if alpha != nil && planeIdx < len(alpha) {
				rgba[rgbaIdx+3] = alpha[planeIdx]
			} else {
				rgba[rgbaIdx+3] = 255
			}
		}
	}

	return rgba
}

// DecodeNSCodecToRGBA decodes an NSCodec bitmap stream to RGBA pixels
func DecodeNSCodecToRGBA(data []byte, width, height int) []byte {
	if len(data) < 20 { // Minimum header size
		return nil
	}

	// Parse header
	lumaPlaneByteCount := int(uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24)
	orangeChromaPlaneByteCount := int(uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24)
	greenChromaPlaneByteCount := int(uint32(data[8]) | uint32(data[9])<<8 | uint32(data[10])<<16 | uint32(data[11])<<24)
	alphaPlaneByteCount := int(uint32(data[12]) | uint32(data[13])<<8 | uint32(data[14])<<16 | uint32(data[15])<<24)
	colorLossLevel := data[16]
	chromaSubsamplingLevel := data[17]
	// Reserved: data[18:20]

	if colorLossLevel < 1 || colorLossLevel > 7 {
		return nil
	}

	chromaSubsampling := chromaSubsamplingLevel != 0

	// Calculate expected plane sizes
	var lumaWidth, lumaHeight int
	var chromaWidth, chromaHeight int

	if chromaSubsampling {
		lumaWidth = roundUpToMultiple(width, 8)
		lumaHeight = height
		chromaWidth = lumaWidth / 2
		chromaHeight = roundUpToMultiple(height, 2) / 2
	} else {
		lumaWidth = width
		lumaHeight = height
		chromaWidth = width
		chromaHeight = height
	}

	lumaExpectedSize := lumaWidth * lumaHeight
	chromaExpectedSize := chromaWidth * chromaHeight

	// Extract planes from data
	offset := 20

	var lumaPlaneData, orangePlaneData, greenPlaneData, alphaPlaneData []byte

	if lumaPlaneByteCount > 0 {
		if offset+lumaPlaneByteCount > len(data) {
			return nil
		}
		lumaPlaneData = data[offset : offset+lumaPlaneByteCount]
		offset += lumaPlaneByteCount
	}

	if orangeChromaPlaneByteCount > 0 {
		if offset+orangeChromaPlaneByteCount > len(data) {
			return nil
		}
		orangePlaneData = data[offset : offset+orangeChromaPlaneByteCount]
		offset += orangeChromaPlaneByteCount
	}

	if greenChromaPlaneByteCount > 0 {
		if offset+greenChromaPlaneByteCount > len(data) {
			return nil
		}
		greenPlaneData = data[offset : offset+greenChromaPlaneByteCount]
		offset += greenChromaPlaneByteCount
	}

	if alphaPlaneByteCount > 0 {
		if offset+alphaPlaneByteCount > len(data) {
			return nil
		}
		alphaPlaneData = data[offset : offset+alphaPlaneByteCount]
	}

	// Decompress planes
	lumaPlane := NSCodecRLEDecompress(lumaPlaneData, lumaExpectedSize)
	if lumaPlane == nil {
		return nil
	}

	orangePlane := NSCodecRLEDecompress(orangePlaneData, chromaExpectedSize)
	if orangePlane == nil {
		return nil
	}

	greenPlane := NSCodecRLEDecompress(greenPlaneData, chromaExpectedSize)
	if greenPlane == nil {
		return nil
	}

	var alphaPlane []byte
	if alphaPlaneByteCount > 0 {
		alphaExpectedSize := width * height
		alphaPlane = NSCodecRLEDecompress(alphaPlaneData, alphaExpectedSize)
		if alphaPlane == nil {
			return nil
		}
	}

	// Apply chroma super-sampling if needed
	if chromaSubsampling {
		orangePlane = ChromaSuperSample(orangePlane, chromaWidth, chromaHeight, lumaWidth, lumaHeight)
		greenPlane = ChromaSuperSample(greenPlane, chromaWidth, chromaHeight, lumaWidth, lumaHeight)
	}

	// Apply color loss restoration
	if colorLossLevel > 1 {
		orangePlane = RestoreColorLoss(orangePlane, colorLossLevel)
		greenPlane = RestoreColorLoss(greenPlane, colorLossLevel)
	}

	// Convert AYCoCg to RGBA
	return AYCoCgToRGBA(lumaPlane, orangePlane, greenPlane, alphaPlane, lumaWidth, lumaHeight, width, height)
}
