package codec

import (
	"errors"
	"fmt"
)

// PixelFormat describes the byte layout for 32-bpp input pixels accepted by
// reusable encoder helpers.
type PixelFormat int

const (
	PixelFormatRGBA PixelFormat = iota + 1
	PixelFormatBGRA
)

const (
	PlanarFlagRLE     byte = 0x10
	PlanarFlagNoAlpha byte = 0x20
)

// PlanarInput describes one top-down 32-bpp image to encode as MS-RDPEGFX
// Planar no-alpha RLE payload data.
type PlanarInput struct {
	Pixels []byte
	Width  int
	Height int
	Stride int
	Format PixelFormat
}

// EncodePlanarNoAlpha encodes RGBA/BGRA pixels as a Planar no-alpha RLE payload.
func EncodePlanarNoAlpha(input PlanarInput) ([]byte, error) {
	if err := validatePlanarInput(input); err != nil {
		return nil, err
	}
	planeSize := input.Width * input.Height
	out := make([]byte, 0, 1+planeSize*3)
	plane := make([]byte, planeSize)
	return appendPlanarNoAlphaWithPlane(out, input, plane), nil
}

func validatePlanarInput(input PlanarInput) error {
	if input.Width <= 0 || input.Height <= 0 {
		return errors.New("width and height must be positive")
	}
	if input.Width > int(^uint(0)>>1)/input.Height {
		return errors.New("planar dimensions overflow")
	}
	if input.Stride < input.Width*4 {
		return errors.New("stride is too small for width")
	}
	if input.Height > 0 && len(input.Pixels) < input.Stride*(input.Height-1)+input.Width*4 {
		return errors.New("pixel buffer is too short for geometry")
	}
	switch input.Format {
	case PixelFormatRGBA, PixelFormatBGRA:
		return nil
	default:
		return fmt.Errorf("unsupported pixel format %d", input.Format)
	}
}

func appendPlanarNoAlphaWithPlane(out []byte, input PlanarInput, plane []byte) []byte {
	out = append(out, PlanarFlagRLE|PlanarFlagNoAlpha)
	for _, component := range []byte{'r', 'g', 'b'} {
		fillPlanarPlane(plane, input, component)
		out = appendPlanarDeltaRLEPlane(out, plane, input.Width, input.Height)
	}
	return out
}

func fillPlanarPlane(plane []byte, input PlanarInput, component byte) {
	for y := 0; y < input.Height; y++ {
		for x := 0; x < input.Width; x++ {
			si := y*input.Stride + x*4
			di := y*input.Width + x
			switch input.Format {
			case PixelFormatRGBA:
				switch component {
				case 'r':
					plane[di] = input.Pixels[si]
				case 'g':
					plane[di] = input.Pixels[si+1]
				case 'b':
					plane[di] = input.Pixels[si+2]
				}
			case PixelFormatBGRA:
				switch component {
				case 'r':
					plane[di] = input.Pixels[si+2]
				case 'g':
					plane[di] = input.Pixels[si+1]
				case 'b':
					plane[di] = input.Pixels[si]
				}
			}
		}
	}
}

func appendPlanarDeltaRLEPlane(out, plane []byte, width, height int) []byte {
	for y := height - 1; y > 0; y-- {
		row := plane[y*width : y*width+width]
		prev := plane[(y-1)*width : (y-1)*width+width]
		for x := 0; x < width; x++ {
			row[x] = PlanarDeltaByte(int(row[x]) - int(prev[x]))
		}
	}
	for y := 0; y < height; y++ {
		out = appendPlanarRLELine(out, plane[y*width:y*width+width])
	}
	return out
}

// PlanarDeltaByte maps a signed delta to the byte representation used by the
// Planar codec's delta transform.
func PlanarDeltaByte(delta int) byte {
	if delta > 127 {
		delta -= 256
	} else if delta < -128 {
		delta += 256
	}
	if delta >= 0 {
		return byte(delta << 1)
	}
	return byte(((-delta) << 1) - 1)
}

func appendPlanarRLELine(out []byte, row []byte) []byte {
	for x := 0; x < len(row); {
		if row[x] == 0 {
			run := 1
			for x+run < len(row) && row[x+run] == 0 && run < 47 {
				run++
			}
			if run >= 16 {
				if run < 32 {
					out = append(out, byte(((run-16)&0x0f)<<4|0x01))
				} else {
					out = append(out, byte(((run-32)&0x0f)<<4|0x02))
				}
				x += run
				continue
			}
		}
		rawStart := x
		rawLen := 0
		for x < len(row) && rawLen < 15 {
			if row[x] == 0 {
				run := 1
				for x+run < len(row) && row[x+run] == 0 && run < 16 {
					run++
				}
				if run >= 16 {
					break
				}
			}
			x++
			rawLen++
		}
		out = append(out, byte(rawLen<<4))
		out = append(out, row[rawStart:rawStart+rawLen]...)
	}
	return out
}
