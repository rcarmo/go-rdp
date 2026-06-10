package codec

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Bitmap Update and RLE helpers follow the classic bitmap update structures in
// MS-RDPBCGR and the interleaved RLE bitmap compression semantics documented in
// MS-RDPBCGR 2.2.9.1.1.3.1.2.4. The encoder is deliberately conservative: it
// emits only COLOR and COPY-style orders that the existing go-rdp decoders can
// round-trip, and callers should only use the compressed envelope when the
// encoded payload is smaller than the raw rectangle.

const (
	UpdateTypeBitmap uint16 = 0x0001

	BitmapCompressionFlag     uint16 = 0x0001
	NoBitmapCompressionHeader uint16 = 0x0400

	BitmapBPP8  uint16 = 8
	BitmapBPP15 uint16 = 15
	BitmapBPP16 uint16 = 16
	BitmapBPP24 uint16 = 24

	bitmapRLEMaxLiteralPixels = 0xffff
)

// BitmapUpdateRect is one TS_BITMAP_DATA rectangle in a Bitmap Update PDU.
type BitmapUpdateRect struct {
	Left, Top, Right, Bottom uint16
	Width, Height            uint16
	BitsPerPixel             uint16
	Flags                    uint16
	Data                     []byte
}

// AlignedBitmapRowBytes returns the DWORD-aligned row size for classic Bitmap Update data.
func AlignedBitmapRowBytes(width int, bitsPerPixel uint16) (int, bool) {
	if width <= 0 {
		return 0, false
	}
	bpp := int(bitsPerPixel)
	if bitsPerPixel == BitmapBPP15 {
		bpp = 16
	}
	switch bitsPerPixel {
	case BitmapBPP8, BitmapBPP15, BitmapBPP16, BitmapBPP24:
	default:
		return 0, false
	}
	bits := width * bpp
	return ((bits + 31) / 32) * 4, true
}

// BitmapBytesPerPixel returns the visible bytes per pixel for classic bitmap formats.
func BitmapBytesPerPixel(bitsPerPixel uint16) (int, bool) {
	switch bitsPerPixel {
	case BitmapBPP8:
		return 1, true
	case BitmapBPP15, BitmapBPP16:
		return 2, true
	case BitmapBPP24:
		return 3, true
	default:
		return 0, false
	}
}

// BuildBitmapUpdate builds a Bitmap Update payload containing one or more rectangles.
func BuildBitmapUpdate(rects []BitmapUpdateRect) ([]byte, error) {
	if len(rects) == 0 {
		return nil, errors.New("Bitmap Update requires at least one rectangle")
	}
	if len(rects) > int(^uint16(0)) {
		return nil, errors.New("too many Bitmap Update rectangles")
	}
	capHint := 4
	for i := range rects {
		if err := validateBitmapUpdateRect(rects[i]); err != nil {
			return nil, fmt.Errorf("rectangle %d: %w", i, err)
		}
		capHint += 18 + len(rects[i].Data)
	}
	out := make([]byte, 0, capHint)
	out = binary.LittleEndian.AppendUint16(out, UpdateTypeBitmap)
	out = binary.LittleEndian.AppendUint16(out, uint16(len(rects))) // #nosec G115 bounded above.
	for _, rect := range rects {
		out = appendBitmapUpdateRect(out, rect)
	}
	return out, nil
}

// BuildCompressedBitmapRLEUpdate builds a Bitmap Update payload whose rectangles
// carry conservative classic RLE streams with BITMAP_COMPRESSION and
// NO_BITMAP_COMPRESSION_HDR set. It rejects inputs that would expand relative to
// the provided raw padded rectangle data.
func BuildCompressedBitmapRLEUpdate(rects []BitmapUpdateRect) ([]byte, error) {
	if len(rects) == 0 {
		return nil, errors.New("Bitmap Update requires at least one rectangle")
	}
	compressed := make([]BitmapUpdateRect, 0, len(rects))
	for i, rect := range rects {
		if err := validateBitmapUpdateRect(rect); err != nil {
			return nil, fmt.Errorf("rectangle %d: %w", i, err)
		}
		encoded, err := EncodeBitmapRLECopy(rect.Data, int(rect.Width), int(rect.Height), rect.BitsPerPixel)
		if err != nil {
			return nil, fmt.Errorf("rectangle %d: %w", i, err)
		}
		if len(encoded) == 0 || len(encoded) >= len(rect.Data) || len(encoded) > int(^uint16(0)) {
			return nil, fmt.Errorf("rectangle %d: compressed payload expands or exceeds 16-bit length", i)
		}
		rect.Flags = BitmapCompressionFlag | NoBitmapCompressionHeader
		rect.Data = encoded
		compressed = append(compressed, rect)
	}
	return BuildBitmapUpdate(compressed)
}

// EncodeBitmapRLECopy encodes padded top-down 8/15/16/24-bpp bitmap data as a
// conservative interleaved RLE stream. Output rows are bottom-up and omit scanline padding.
func EncodeBitmapRLECopy(data []byte, width, height int, bitsPerPixel uint16) ([]byte, error) {
	bytesPerPixel, ok := BitmapBytesPerPixel(bitsPerPixel)
	if !ok {
		return nil, fmt.Errorf("unsupported bitsPerPixel %d", bitsPerPixel)
	}
	if width <= 0 || height <= 0 {
		return nil, errors.New("width and height must be positive")
	}
	rowBytes, ok := AlignedBitmapRowBytes(width, bitsPerPixel)
	if !ok {
		return nil, fmt.Errorf("unsupported bitsPerPixel %d", bitsPerPixel)
	}
	visibleRowBytes := width * bytesPerPixel
	required := rowBytes * height
	if visibleRowBytes <= 0 || rowBytes < visibleRowBytes || len(data) < required {
		return nil, errors.New("bitmap data is too short for geometry")
	}
	if pixel, ok := bitmapRLESolidPixel(data, width, height, rowBytes, visibleRowBytes, bytesPerPixel); ok {
		out := make([]byte, 0, bitmapRLEColorOrderLen(width, bytesPerPixel)*height)
		for y := 0; y < height; y++ {
			out = appendBitmapRLEColorOrder(out, width, pixel)
		}
		return out, nil
	}
	out := make([]byte, 0, visibleRowBytes*height+height)
	for y := height - 1; y >= 0; y-- {
		row := data[y*rowBytes : y*rowBytes+visibleRowBytes]
		for offset := 0; offset < len(row); {
			pixels := (len(row) - offset) / bytesPerPixel
			if pixels > bitmapRLEMaxLiteralPixels {
				pixels = bitmapRLEMaxLiteralPixels
			}
			span := row[offset : offset+pixels*bytesPerPixel]
			out = appendBitmapRLECopyOrder(out, pixels, bytesPerPixel, span)
			offset += pixels * bytesPerPixel
		}
	}
	return out, nil
}

func validateBitmapUpdateRect(rect BitmapUpdateRect) error {
	if rect.Width == 0 || rect.Height == 0 {
		return errors.New("width and height must be non-zero")
	}
	if rect.Right < rect.Left || rect.Bottom < rect.Top {
		return errors.New("invalid destination rectangle")
	}
	if rect.Right-rect.Left+1 != rect.Width || rect.Bottom-rect.Top+1 != rect.Height {
		return errors.New("destination rectangle does not match width/height")
	}
	if _, ok := BitmapBytesPerPixel(rect.BitsPerPixel); !ok {
		return fmt.Errorf("unsupported bitsPerPixel %d", rect.BitsPerPixel)
	}
	if len(rect.Data) == 0 {
		return errors.New("bitmap data must be non-empty")
	}
	if len(rect.Data) > int(^uint16(0)) {
		return errors.New("bitmap data exceeds 16-bit length")
	}
	return nil
}

func appendBitmapUpdateRect(out []byte, rect BitmapUpdateRect) []byte {
	out = binary.LittleEndian.AppendUint16(out, rect.Left)
	out = binary.LittleEndian.AppendUint16(out, rect.Top)
	out = binary.LittleEndian.AppendUint16(out, rect.Right)
	out = binary.LittleEndian.AppendUint16(out, rect.Bottom)
	out = binary.LittleEndian.AppendUint16(out, rect.Width)
	out = binary.LittleEndian.AppendUint16(out, rect.Height)
	out = binary.LittleEndian.AppendUint16(out, rect.BitsPerPixel)
	out = binary.LittleEndian.AppendUint16(out, rect.Flags)
	out = binary.LittleEndian.AppendUint16(out, uint16(len(rect.Data))) // #nosec G115 validated above.
	return append(out, rect.Data...)
}

func appendBitmapRLECopyOrder(out []byte, pixels, bytesPerPixel int, data []byte) []byte {
	if pixels <= 0 || bytesPerPixel <= 0 || len(data) < pixels*bytesPerPixel {
		return out
	}
	if bitmapRLEAllSamePixel(data, pixels, bytesPerPixel) {
		return appendBitmapRLEColorOrder(out, pixels, data[:bytesPerPixel])
	}
	if pixels < 32 {
		out = append(out, byte(0x80|pixels))
	} else if pixels < 32+256 {
		out = append(out, 0x80, byte(pixels-32))
	} else {
		out = append(out, 0xf4)
		out = binary.LittleEndian.AppendUint16(out, uint16(pixels)) // #nosec G115 bounded by bitmapRLEMaxLiteralPixels.
	}
	return append(out, data[:pixels*bytesPerPixel]...)
}

func bitmapRLESolidPixel(data []byte, width, height, rowBytes, visibleRowBytes, bytesPerPixel int) ([]byte, bool) {
	if len(data) < rowBytes*height || visibleRowBytes < bytesPerPixel || width <= 0 || height <= 0 {
		return nil, false
	}
	first := data[:bytesPerPixel]
	for y := 0; y < height; y++ {
		row := data[y*rowBytes : y*rowBytes+visibleRowBytes]
		for x := 0; x < width; x++ {
			off := x * bytesPerPixel
			for b := 0; b < bytesPerPixel; b++ {
				if row[off+b] != first[b] {
					return nil, false
				}
			}
		}
	}
	return first, true
}

func bitmapRLEColorOrderLen(pixels, bytesPerPixel int) int {
	switch {
	case pixels < 32:
		return 1 + bytesPerPixel
	case pixels < 32+256:
		return 2 + bytesPerPixel
	default:
		return 3 + bytesPerPixel
	}
}

func appendBitmapRLEColorOrder(out []byte, pixels int, pixel []byte) []byte {
	if pixels <= 0 || len(pixel) == 0 {
		return out
	}
	if pixels < 32 {
		out = append(out, byte(0x60|pixels))
	} else if pixels < 32+256 {
		out = append(out, 0x60, byte(pixels-32))
	} else {
		out = append(out, 0xf3)
		out = binary.LittleEndian.AppendUint16(out, uint16(pixels)) // #nosec G115 bounded by bitmapRLEMaxLiteralPixels.
	}
	return append(out, pixel...)
}

func bitmapRLEAllSamePixel(data []byte, pixels, bytesPerPixel int) bool {
	if pixels <= 1 || bytesPerPixel <= 0 || len(data) < pixels*bytesPerPixel {
		return false
	}
	first := data[:bytesPerPixel]
	for i := 1; i < pixels; i++ {
		o := i * bytesPerPixel
		for b := 0; b < bytesPerPixel; b++ {
			if data[o+b] != first[b] {
				return false
			}
		}
	}
	return true
}
