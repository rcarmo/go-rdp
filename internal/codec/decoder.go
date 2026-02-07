// Package nscodec implements the NSCodec bitmap codec decoder as specified in MS-RDPNSC.
// NSCodec compresses 24/32 bpp images using AYCoCg color space conversion and RLE compression.
package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrInvalidStream     = errors.New("nscodec: invalid bitmap stream")
	ErrInvalidPlaneSize  = errors.New("nscodec: invalid plane size")
	ErrInvalidColorLoss  = errors.New("nscodec: invalid color loss level")
	ErrDecompressionFail = errors.New("nscodec: RLE decompression failed")
)

// NSCodec GUID: CA8D1BB9-000F-154F-589F-AE2D1A87E2D6
var NSCodecGUID = [16]byte{
	0xB9, 0x1B, 0x8D, 0xCA, 0x0F, 0x00, 0x4F, 0x15,
	0x58, 0x9F, 0xAE, 0x2D, 0x1A, 0x87, 0xE2, 0xD6,
}

// BitmapStream represents the NSCODEC_BITMAP_STREAM structure as specified in MS-RDPNSC.
type BitmapStream struct {
	LumaPlaneByteCount         uint32
	OrangeChromaPlaneByteCount uint32
	GreenChromaPlaneByteCount  uint32
	AlphaPlaneByteCount        uint32
	ColorLossLevel             uint8
	ChromaSubsamplingLevel     uint8
	LumaPlane                  []byte
	OrangeChromaPlane          []byte
	GreenChromaPlane           []byte
	AlphaPlane                 []byte
}

// ParseBitmapStream parses an NSCODEC_BITMAP_STREAM from raw bytes.
func ParseBitmapStream(data []byte) (*BitmapStream, error) {
	if len(data) < 18 { // Minimum header size
		return nil, ErrInvalidStream
	}

	stream := &BitmapStream{
		LumaPlaneByteCount:         binary.LittleEndian.Uint32(data[0:4]),
		OrangeChromaPlaneByteCount: binary.LittleEndian.Uint32(data[4:8]),
		GreenChromaPlaneByteCount:  binary.LittleEndian.Uint32(data[8:12]),
		AlphaPlaneByteCount:        binary.LittleEndian.Uint32(data[12:16]),
		ColorLossLevel:             data[16],
		ChromaSubsamplingLevel:     data[17],
		// Reserved: data[18:20]
	}

	if stream.ColorLossLevel < 1 || stream.ColorLossLevel > 7 {
		return nil, ErrInvalidColorLoss
	}

	offset := uint32(20) // Header size
	dataLen := uint32(len(data)) // #nosec G115

	// Parse luma plane
	if stream.LumaPlaneByteCount > 0 {
		if dataLen < offset+stream.LumaPlaneByteCount {
			return nil, ErrInvalidPlaneSize
		}
		stream.LumaPlane = data[offset : offset+stream.LumaPlaneByteCount]
		offset += stream.LumaPlaneByteCount
	}

	// Parse orange chroma plane
	if stream.OrangeChromaPlaneByteCount > 0 {
		if dataLen < offset+stream.OrangeChromaPlaneByteCount {
			return nil, ErrInvalidPlaneSize
		}
		stream.OrangeChromaPlane = data[offset : offset+stream.OrangeChromaPlaneByteCount]
		offset += stream.OrangeChromaPlaneByteCount
	}

	// Parse green chroma plane
	if stream.GreenChromaPlaneByteCount > 0 {
		if dataLen < offset+stream.GreenChromaPlaneByteCount {
			return nil, ErrInvalidPlaneSize
		}
		stream.GreenChromaPlane = data[offset : offset+stream.GreenChromaPlaneByteCount]
		offset += stream.GreenChromaPlaneByteCount
	}

	// Parse alpha plane (optional)
	if stream.AlphaPlaneByteCount > 0 {
		if dataLen < offset+stream.AlphaPlaneByteCount {
			return nil, ErrInvalidPlaneSize
		}
		stream.AlphaPlane = data[offset : offset+stream.AlphaPlaneByteCount]
	}

	return stream, nil
}

// Decode decodes an NSCodec bitmap stream to RGBA pixels.
// Width and height specify the image dimensions. Returns RGBA pixel data (4 bytes per pixel).
func Decode(data []byte, width, height int) ([]byte, error) {
	stream, err := ParseBitmapStream(data)
	if err != nil {
		return nil, err
	}

	return stream.Decode(width, height)
}

// Decode decodes the bitmap stream to RGBA pixels
func (s *BitmapStream) Decode(width, height int) ([]byte, error) {
	chromaSubsampling := s.ChromaSubsamplingLevel != 0

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

	// Decompress or use raw luma plane
	lumaPlane, err := decompressPlane(s.LumaPlane, lumaExpectedSize)
	if err != nil {
		return nil, fmt.Errorf("luma plane: %w", err)
	}

	// Decompress or use raw orange chroma plane
	orangePlane, err := decompressPlane(s.OrangeChromaPlane, chromaExpectedSize)
	if err != nil {
		return nil, fmt.Errorf("orange chroma plane: %w", err)
	}

	// Decompress or use raw green chroma plane
	greenPlane, err := decompressPlane(s.GreenChromaPlane, chromaExpectedSize)
	if err != nil {
		return nil, fmt.Errorf("green chroma plane: %w", err)
	}

	// Decompress or use raw alpha plane (if present)
	var alphaPlane []byte
	if s.AlphaPlaneByteCount > 0 {
		alphaExpectedSize := width * height
		alphaPlane, err = decompressPlane(s.AlphaPlane, alphaExpectedSize)
		if err != nil {
			return nil, fmt.Errorf("alpha plane: %w", err)
		}
	}

	// Apply chroma super-sampling if needed
	if chromaSubsampling {
		orangePlane = chromaSuperSample(orangePlane, chromaWidth, chromaHeight, lumaWidth, lumaHeight)
		greenPlane = chromaSuperSample(greenPlane, chromaWidth, chromaHeight, lumaWidth, lumaHeight)
	}

	// Apply color loss restoration
	if s.ColorLossLevel > 1 {
		orangePlane = restoreColorLoss(orangePlane, s.ColorLossLevel)
		greenPlane = restoreColorLoss(greenPlane, s.ColorLossLevel)
	}

	// Convert AYCoCg to RGBA
	return aycoCgToRGBA(lumaPlane, orangePlane, greenPlane, alphaPlane, lumaWidth, lumaHeight, width, height)
}

// decompressPlane decompresses an RLE-compressed plane or returns raw data
func decompressPlane(data []byte, expectedSize int) ([]byte, error) {
	if len(data) == expectedSize {
		// Raw data, no decompression needed
		return data, nil
	}

	if len(data) > expectedSize {
		return nil, ErrInvalidPlaneSize
	}

	// RLE compressed - decompress
	return rleDecompress(data, expectedSize)
}

// rleDecompress decompresses NSCodec RLE data
// NSCodec RLE format: segments followed by 4-byte EndData (last 4 raw bytes)
func rleDecompress(data []byte, expectedSize int) ([]byte, error) {
	if len(data) < 4 {
		return nil, ErrDecompressionFail
	}

	result := make([]byte, 0, expectedSize)
	offset := 0
	dataLen := len(data) - 4 // Exclude EndData

	for offset < dataLen && len(result) < expectedSize-4 {
		if offset >= dataLen {
			break
		}

		// Read segment header
		header := data[offset]
		offset++

		if header&0x80 != 0 {
			// Run segment: repeat single byte
			runLength := int(header & 0x7F)
			if runLength == 0 {
				// Extended run length
				if offset >= dataLen {
					return nil, ErrDecompressionFail
				}
				runLength = int(data[offset]) + 128
				offset++
			}
			if offset >= dataLen {
				return nil, ErrDecompressionFail
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
				// Extended literal length
				if offset >= dataLen {
					return nil, ErrDecompressionFail
				}
				literalLength = int(data[offset]) + 128
				offset++
			}

			if offset+literalLength > dataLen {
				return nil, ErrDecompressionFail
			}

			result = append(result, data[offset:offset+literalLength]...)
			offset += literalLength
		}
	}

	// Append EndData (last 4 bytes of original plane)
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

	return result[:expectedSize], nil
}

// chromaSuperSample upsamples chroma planes from subsampled to full resolution
func chromaSuperSample(plane []byte, srcWidth, srcHeight, dstWidth, dstHeight int) []byte {
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

// restoreColorLoss restores color values that were quantized during compression
func restoreColorLoss(plane []byte, colorLossLevel uint8) []byte {
	if colorLossLevel <= 1 {
		return plane
	}

	// Color loss shifts values, we need to restore them
	shift := colorLossLevel - 1
	result := make([]byte, len(plane))

	for i, v := range plane {
		// Restore by left-shifting and adding mid-value
		restored := int(v) << shift
		if restored > 255 {
			restored = 255
		}
		result[i] = byte(restored)
	}

	return result
}

// aycoCgToRGBA converts AYCoCg color space to RGBA
// Y = Luma, Co = Orange Chroma, Cg = Green Chroma
func aycoCgToRGBA(luma, co, cg, alpha []byte, planeWidth, planeHeight, imgWidth, imgHeight int) ([]byte, error) {
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
			// t = Y - Cg
			// R = t + Co
			// G = Y + Cg
			// B = t - Co
			t := yVal - cgVal
			r := clamp(t + coVal)
			g := clamp(yVal + cgVal)
			b := clamp(t - coVal)

			rgba[rgbaIdx+0] = byte(r)
			rgba[rgbaIdx+1] = byte(g)
			rgba[rgbaIdx+2] = byte(b)

			// Alpha
			if alpha != nil && planeIdx < len(alpha) {
				rgba[rgbaIdx+3] = alpha[planeIdx]
			} else {
				rgba[rgbaIdx+3] = 255
			}
		}
	}

	return rgba, nil
}

// clamp clamps a value to 0-255 range
func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// roundUpToMultiple rounds n up to the nearest multiple of m
func roundUpToMultiple(n, m int) int {
	if m == 0 {
		return n
	}
	remainder := n % m
	if remainder == 0 {
		return n
	}
	return n + m - remainder
}
