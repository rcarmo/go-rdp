package codec

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"testing"
)

func TestEncodeJPEGSurfaceBitsPayloadHeadersAndDimensions(t *testing.T) {
	payload, err := EncodeJPEGSurfaceBitsPayload(BitmapInput{
		Pixels: []byte{
			0xff, 0x00, 0x00, 0xff, 0x00, 0xff, 0x00, 0xff,
			0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
		},
		Width: 2, Height: 2, Stride: 8, Format: PixelFormatRGBA,
	}, 80)
	if err != nil {
		t.Fatalf("EncodeJPEGSurfaceBitsPayload: %v", err)
	}
	if len(payload) < 4 || payload[0] != 0xff || payload[1] != 0xd8 || payload[len(payload)-2] != 0xff || payload[len(payload)-1] != 0xd9 {
		t.Fatalf("payload does not have JPEG SOI/EOI markers: %x", payload)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("DecodeConfig: %v", err)
	}
	if format != "jpeg" || cfg.Width != 2 || cfg.Height != 2 {
		t.Fatalf("DecodeConfig = format=%q %dx%d", format, cfg.Width, cfg.Height)
	}
	img, err := jpeg.Decode(bytes.NewReader(payload))
	if err != nil || img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
		t.Fatalf("jpeg.Decode bounds=%v err=%v", img.Bounds(), err)
	}
}

func TestEncodeJPEGSurfaceBitsPayloadDeterministic(t *testing.T) {
	input := BitmapInput{Pixels: []byte{
		0x30, 0x20, 0x10, 0xaa, 0x60, 0x50, 0x40, 0xbb,
		0x90, 0x80, 0x70, 0xcc, 0xc0, 0xb0, 0xa0, 0xdd,
	}, Width: 2, Height: 2, Stride: 8, Format: PixelFormatBGRA}
	first, err := EncodeJPEGSurfaceBitsPayload(input, 75)
	if err != nil {
		t.Fatalf("EncodeJPEGSurfaceBitsPayload first: %v", err)
	}
	second, err := EncodeJPEGSurfaceBitsPayload(input, 75)
	if err != nil {
		t.Fatalf("EncodeJPEGSurfaceBitsPayload second: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("JPEG output is not deterministic for identical input")
	}
}

func TestEncodeJPEGSurfaceBitsPayloadInputFormats(t *testing.T) {
	cases := []struct {
		name   string
		pixels []byte
		stride int
		format PixelFormat
	}{
		{"rgba", []byte{0x10, 0x20, 0x30, 0xff}, 4, PixelFormatRGBA},
		{"bgra", []byte{0x30, 0x20, 0x10, 0xff}, 4, PixelFormatBGRA},
		{"bgr", []byte{0x30, 0x20, 0x10}, 3, PixelFormatBGR},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := EncodeJPEGSurfaceBitsPayload(BitmapInput{Pixels: tc.pixels, Width: 1, Height: 1, Stride: tc.stride, Format: tc.format}, 90)
			if err != nil {
				t.Fatalf("EncodeJPEGSurfaceBitsPayload: %v", err)
			}
			cfg, format, err := image.DecodeConfig(bytes.NewReader(payload))
			if err != nil || format != "jpeg" || cfg.Width != 1 || cfg.Height != 1 {
				t.Fatalf("DecodeConfig format=%q cfg=%+v err=%v", format, cfg, err)
			}
		})
	}
}

func TestEncodeJPEGSurfaceBitsPayloadRejectsInvalid(t *testing.T) {
	valid := BitmapInput{Pixels: []byte{0, 0, 0, 0}, Width: 1, Height: 1, Stride: 4, Format: PixelFormatRGBA}
	for _, quality := range []int{0, -1, 101} {
		if _, err := EncodeJPEGSurfaceBitsPayload(valid, quality); err == nil {
			t.Fatalf("EncodeJPEGSurfaceBitsPayload accepted quality %d", quality)
		}
	}
	cases := []BitmapInput{
		{},
		{Pixels: valid.Pixels, Width: 1, Height: 1, Stride: 3, Format: PixelFormatRGBA},
		{Pixels: []byte{0, 0, 0}, Width: 1, Height: 1, Stride: 4, Format: PixelFormatRGBA},
		{Pixels: valid.Pixels, Width: 1, Height: 1, Stride: 4, Format: 99},
	}
	for _, tc := range cases {
		if _, err := EncodeJPEGSurfaceBitsPayload(tc, 75); err == nil {
			t.Fatalf("EncodeJPEGSurfaceBitsPayload(%#v) err = nil", tc)
		}
	}
}

func TestBuildJPEGSetSurfaceBitsEnvelope(t *testing.T) {
	input := BitmapInput{Pixels: []byte{0x10, 0x20, 0x30, 0xff}, Width: 1, Height: 1, Stride: 4, Format: PixelFormatRGBA}
	cmd, err := BuildJPEGSetSurfaceBits(Rect{Left: 4, Top: 5, Right: 4, Bottom: 5}, 0x12, input, 80)
	if err != nil {
		t.Fatalf("BuildJPEGSetSurfaceBits: %v", err)
	}
	if binary.LittleEndian.Uint16(cmd[0:2]) != SurfaceCmdSetSurfaceBits {
		t.Fatalf("cmdType = 0x%04x", binary.LittleEndian.Uint16(cmd[0:2]))
	}
	if cmd[12] != 0 || cmd[13] != 0x12 || binary.LittleEndian.Uint16(cmd[14:16]) != 1 || binary.LittleEndian.Uint16(cmd[16:18]) != 1 {
		t.Fatalf("unexpected SurfaceBits header: %x", cmd[:22])
	}
	bitmapLen := int(binary.LittleEndian.Uint32(cmd[18:22]))
	if bitmapLen != len(cmd)-22 || bitmapLen == 0 {
		t.Fatalf("bitmapLen=%d total=%d", bitmapLen, len(cmd))
	}
	if _, format, err := image.DecodeConfig(bytes.NewReader(cmd[22:])); err != nil || format != "jpeg" {
		t.Fatalf("SurfaceBits payload format=%q err=%v", format, err)
	}
}
