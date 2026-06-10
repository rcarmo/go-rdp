package codec

import (
	"encoding/binary"
	"errors"
)

const (
	ClearCodecMagic            = "CLR0"
	ClearCodecOpSolidRect byte = 1
	ClearCodecOpRawRect   byte = 2

	clearCodecMaxRawRectBytes = 256 * 1024
	clearCodecTileSize        = 64
	clearCodecMaxPayloadLen   = 1024 * 1024
)

var ErrClearCodecEncodeFailed = errors.New("clearcodec: encode failed")

// EncodeClearCodecMinimal encodes RGBA/BGRA pixels using the explicitly limited
// ClearCodec subset supported by this package: solid rectangles and RGB565 raw
// rectangles split into 64x64 tiles. The helper rejects payloads that would be
// empty, exceed bounded payload limits, or expand relative to 32-bpp input.
func EncodeClearCodecMinimal(input BitmapInput) ([]byte, error) {
	stride, ok := clearCodecInputStride(input)
	if !ok || input.Width > 8192 || input.Height > 8192 {
		return nil, ErrClearCodecEncodeFailed
	}
	rawBytes := input.Width * input.Height * 4
	if rawBytes <= 0 || rawBytes > clearCodecMaxPayloadLen {
		return nil, ErrClearCodecEncodeFailed
	}
	if r, g, b, solid := clearCodecSolidRGB(input, stride); solid {
		payload := make([]byte, 22)
		copy(payload[0:4], ClearCodecMagic)
		binary.LittleEndian.PutUint16(payload[4:6], uint16(input.Width))
		binary.LittleEndian.PutUint16(payload[6:8], uint16(input.Height))
		binary.LittleEndian.PutUint16(payload[8:10], 1)
		payload[10] = ClearCodecOpSolidRect
		binary.LittleEndian.PutUint16(payload[15:17], uint16(input.Width))
		binary.LittleEndian.PutUint16(payload[17:19], uint16(input.Height))
		payload[19], payload[20], payload[21] = r, g, b
		if len(payload) >= rawBytes || len(payload) > clearCodecMaxPayloadLen {
			return nil, ErrClearCodecEncodeFailed
		}
		return payload, nil
	}
	payload, ok := buildClearCodecRects(input, stride, rawBytes)
	if !ok {
		return nil, ErrClearCodecEncodeFailed
	}
	return payload, nil
}

// BuildClearCodecWireToSurface encodes a minimal ClearCodec payload and wraps it
// in an RDPGFX WireToSurface_1 PDU.
func BuildClearCodecWireToSurface(surfaceID uint16, pixelFormat uint8, dest Rect, input BitmapInput) ([]byte, error) {
	payload, err := EncodeClearCodecMinimal(input)
	if err != nil {
		return nil, err
	}
	return BuildRDPGFXWireToSurface1(surfaceID, RDPGFXCodecClearCodec, pixelFormat, dest, payload)
}

// RGB565FromRGB converts 8-bit RGB to packed RGB565.
func RGB565FromRGB(r, g, b byte) uint16 {
	return uint16(r>>3)<<11 | uint16(g>>2)<<5 | uint16(b>>3)
}

func clearCodecInputStride(input BitmapInput) (int, bool) {
	if input.Width <= 0 || input.Height <= 0 || input.Stride <= 0 {
		return 0, false
	}
	if input.Width > int(^uint16(0)) || input.Height > int(^uint16(0)) {
		return 0, false
	}
	switch input.Format {
	case PixelFormatRGBA, PixelFormatBGRA:
	default:
		return 0, false
	}
	if input.Stride < input.Width*4 {
		return 0, false
	}
	need := input.Stride*(input.Height-1) + input.Width*4
	if need <= 0 || len(input.Pixels) < need {
		return 0, false
	}
	return input.Stride, true
}

func buildClearCodecRects(input BitmapInput, stride int, rawBytes int) ([]byte, bool) {
	tilesX := (input.Width + clearCodecTileSize - 1) / clearCodecTileSize
	tilesY := (input.Height + clearCodecTileSize - 1) / clearCodecTileSize
	numRects := tilesX * tilesY
	if numRects <= 0 || numRects > int(^uint16(0)) {
		return nil, false
	}
	const headerLen = 10
	maxRectHeaderLen := numRects * (1 + 2 + 2 + 2 + 2 + 4)
	payload := make([]byte, headerLen, headerLen+input.Width*input.Height*2+maxRectHeaderLen)
	copy(payload[0:4], ClearCodecMagic)
	binary.LittleEndian.PutUint16(payload[4:6], uint16(input.Width))
	binary.LittleEndian.PutUint16(payload[6:8], uint16(input.Height))
	rectCount := 0
	for y0 := 0; y0 < input.Height; y0 += clearCodecTileSize {
		h := clearCodecTileSize
		if y0+h > input.Height {
			h = input.Height - y0
		}
		for x0 := 0; x0 < input.Width; x0 += clearCodecTileSize {
			w := clearCodecTileSize
			if x0+w > input.Width {
				w = input.Width - x0
			}
			before := len(payload)
			var ok bool
			payload, ok = appendClearCodecRect(payload, input, stride, x0, y0, w, h)
			if !ok || len(payload) > clearCodecMaxPayloadLen || len(payload) >= rawBytes {
				return nil, false
			}
			if len(payload) > before {
				rectCount++
			}
		}
	}
	if rectCount == 0 || rectCount > int(^uint16(0)) {
		return nil, false
	}
	binary.LittleEndian.PutUint16(payload[8:10], uint16(rectCount))
	return payload, true
}

func appendClearCodecRect(payload []byte, input BitmapInput, stride, x0, y0, w, h int) ([]byte, bool) {
	r, g, b, solid, ok := clearCodecSolidRectRGB(input, stride, x0, y0, w, h)
	if !ok {
		return nil, false
	}
	if solid {
		payload = append(payload, ClearCodecOpSolidRect)
		payload = binary.LittleEndian.AppendUint16(payload, uint16(x0))
		payload = binary.LittleEndian.AppendUint16(payload, uint16(y0))
		payload = binary.LittleEndian.AppendUint16(payload, uint16(w))
		payload = binary.LittleEndian.AppendUint16(payload, uint16(h))
		payload = binary.LittleEndian.AppendUint32(payload, 0)
		return append(payload, r, g, b), true
	}
	rectLen := w * h * 2
	if rectLen <= 0 || rectLen > clearCodecMaxRawRectBytes {
		return nil, false
	}
	payload = append(payload, ClearCodecOpRawRect)
	payload = binary.LittleEndian.AppendUint16(payload, uint16(x0))
	payload = binary.LittleEndian.AppendUint16(payload, uint16(y0))
	payload = binary.LittleEndian.AppendUint16(payload, uint16(w))
	payload = binary.LittleEndian.AppendUint16(payload, uint16(h))
	payload = binary.LittleEndian.AppendUint32(payload, uint32(rectLen))
	for y := 0; y < h; y++ {
		row := (y0 + y) * stride
		for x := 0; x < w; x++ {
			si := row + (x0+x)*4
			var r8, g8, b8 byte
			switch input.Format {
			case PixelFormatRGBA:
				r8, g8, b8 = input.Pixels[si], input.Pixels[si+1], input.Pixels[si+2]
			case PixelFormatBGRA:
				r8, g8, b8 = input.Pixels[si+2], input.Pixels[si+1], input.Pixels[si]
			default:
				return nil, false
			}
			payload = binary.LittleEndian.AppendUint16(payload, RGB565FromRGB(r8, g8, b8))
		}
	}
	return payload, true
}

func clearCodecSolidRGB(input BitmapInput, stride int) (r, g, b byte, solid bool) {
	r, g, b, solid, ok := clearCodecSolidRectRGB(input, stride, 0, 0, input.Width, input.Height)
	if !ok {
		return 0, 0, 0, false
	}
	return r, g, b, solid
}

func clearCodecSolidRectRGB(input BitmapInput, stride, x0, y0, width, height int) (r, g, b byte, solid bool, ok bool) {
	if width <= 0 || height <= 0 || x0 < 0 || y0 < 0 || x0+width > input.Width || y0+height > input.Height {
		return 0, 0, 0, false, false
	}
	firstR, firstG, firstB, ok := clearCodecRGBAt(input, stride, x0, y0)
	if !ok {
		return 0, 0, 0, false, false
	}
	for y := y0; y < y0+height; y++ {
		for x := x0; x < x0+width; x++ {
			r, g, b, ok := clearCodecRGBAt(input, stride, x, y)
			if !ok {
				return 0, 0, 0, false, false
			}
			if r != firstR || g != firstG || b != firstB {
				return firstR, firstG, firstB, false, true
			}
		}
	}
	return firstR, firstG, firstB, true, true
}

func clearCodecRGBAt(input BitmapInput, stride, x, y int) (r, g, b byte, ok bool) {
	si := y*stride + x*4
	switch input.Format {
	case PixelFormatRGBA:
		return input.Pixels[si], input.Pixels[si+1], input.Pixels[si+2], true
	case PixelFormatBGRA:
		return input.Pixels[si+2], input.Pixels[si+1], input.Pixels[si], true
	default:
		return 0, 0, 0, false
	}
}
