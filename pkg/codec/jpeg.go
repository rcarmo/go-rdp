package codec

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
)

var (
	errJPEGQualityOutOfRange = errors.New("jpeg quality must be between 1 and 100")
	errJPEGSurfaceBitsBounds = errors.New("jpeg SurfaceBits dimensions exceed 16-bit bounds")
)

// EncodeJPEGSurfaceBitsPayload encodes RGBA/BGRA/BGR pixels as a baseline JPEG
// payload suitable for the MS-RDPBCGR Bitmap Codecs JPEG Set Surface Bits path.
// Alpha/reserved input bytes are ignored.
func EncodeJPEGSurfaceBitsPayload(input BitmapInput, quality int) ([]byte, error) {
	bytesPerPixel, err := validateBitmapInput(input)
	if err != nil {
		return nil, err
	}
	if err := validateJPEGQuality(quality); err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rect(0, 0, input.Width, input.Height))
	for y := 0; y < input.Height; y++ {
		for x := 0; x < input.Width; x++ {
			si := y*input.Stride + x*bytesPerPixel
			var r, g, b byte
			switch input.Format {
			case PixelFormatRGBA:
				r, g, b = input.Pixels[si], input.Pixels[si+1], input.Pixels[si+2]
			case PixelFormatBGRA:
				r, g, b = input.Pixels[si+2], input.Pixels[si+1], input.Pixels[si]
			case PixelFormatBGR:
				r, g, b = input.Pixels[si+2], input.Pixels[si+1], input.Pixels[si]
			}
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 0xff})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// BuildJPEGSetSurfaceBits encodes pixels as JPEG and wraps the payload in a Set
// Surface Bits command. codecID must be the ID negotiated for JPEG in the Bitmap
// Codecs Capability Set.
func BuildJPEGSetSurfaceBits(dest Rect, codecID uint8, input BitmapInput, quality int) ([]byte, error) {
	if input.Width > int(^uint16(0)) || input.Height > int(^uint16(0)) {
		return nil, errJPEGSurfaceBitsBounds
	}
	payload, err := EncodeJPEGSurfaceBitsPayload(input, quality)
	if err != nil {
		return nil, err
	}
	return BuildSetSurfaceBits(dest, 32, codecID, uint16(input.Width), uint16(input.Height), payload) // #nosec G115 guarded above.
}

func validateJPEGQuality(quality int) error {
	if quality < 1 || quality > 100 {
		return errJPEGQualityOutOfRange
	}
	return nil
}
