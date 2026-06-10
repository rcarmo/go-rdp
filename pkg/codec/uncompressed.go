package codec

import (
	"errors"
	"fmt"
)

const (
	// PixelFormatBGR describes packed 24-bpp BGR input pixels.
	PixelFormatBGR PixelFormat = 3
)

// BitmapInput describes top-down pixels for simple bitmap payload encoders.
type BitmapInput struct {
	Pixels []byte
	Width  int
	Height int
	Stride int
	Format PixelFormat
}

// EncodeRDPGFXUncompressed encodes pixels as the 32-bpp XRGB byte stream used by
// RDPGFX Uncompressed WireToSurface payloads. Output pixels are in BGRA/XRGB
// memory order with an opaque alpha/reserved byte.
func EncodeRDPGFXUncompressed(input BitmapInput) ([]byte, error) {
	bytesPerPixel, err := validateBitmapInput(input)
	if err != nil {
		return nil, err
	}
	out := make([]byte, input.Width*input.Height*4)
	for y := 0; y < input.Height; y++ {
		for x := 0; x < input.Width; x++ {
			si := y*input.Stride + x*bytesPerPixel
			di := (y*input.Width + x) * 4
			switch input.Format {
			case PixelFormatRGBA:
				out[di+0] = input.Pixels[si+2]
				out[di+1] = input.Pixels[si+1]
				out[di+2] = input.Pixels[si+0]
			case PixelFormatBGRA:
				out[di+0] = input.Pixels[si+0]
				out[di+1] = input.Pixels[si+1]
				out[di+2] = input.Pixels[si+2]
			case PixelFormatBGR:
				out[di+0] = input.Pixels[si+0]
				out[di+1] = input.Pixels[si+1]
				out[di+2] = input.Pixels[si+2]
			}
			out[di+3] = 0xff
		}
	}
	return out, nil
}

func validateBitmapInput(input BitmapInput) (int, error) {
	if input.Width <= 0 || input.Height <= 0 {
		return 0, errors.New("width and height must be positive")
	}
	if input.Width > int(^uint(0)>>1)/input.Height || input.Width*input.Height > int(^uint(0)>>1)/4 {
		return 0, errors.New("bitmap dimensions overflow")
	}
	var bytesPerPixel int
	switch input.Format {
	case PixelFormatRGBA, PixelFormatBGRA:
		bytesPerPixel = 4
	case PixelFormatBGR:
		bytesPerPixel = 3
	default:
		return 0, fmt.Errorf("unsupported pixel format %d", input.Format)
	}
	if input.Stride < input.Width*bytesPerPixel {
		return 0, errors.New("stride is too small for width")
	}
	if len(input.Pixels) < input.Stride*(input.Height-1)+input.Width*bytesPerPixel {
		return 0, errors.New("pixel buffer is too short for geometry")
	}
	return bytesPerPixel, nil
}
