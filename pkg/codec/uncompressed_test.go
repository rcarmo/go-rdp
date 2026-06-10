package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestEncodeRDPGFXUncompressedConversions(t *testing.T) {
	want := []byte{
		0x30, 0x20, 0x10, 0xff,
		0x60, 0x50, 0x40, 0xff,
	}
	cases := []struct {
		name   string
		pixels []byte
		stride int
		format PixelFormat
	}{
		{"rgba", []byte{0x10, 0x20, 0x30, 0xaa, 0x40, 0x50, 0x60, 0xbb}, 8, PixelFormatRGBA},
		{"bgra", []byte{0x30, 0x20, 0x10, 0xaa, 0x60, 0x50, 0x40, 0xbb}, 8, PixelFormatBGRA},
		{"bgr", []byte{0x30, 0x20, 0x10, 0x60, 0x50, 0x40}, 6, PixelFormatBGR},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EncodeRDPGFXUncompressed(BitmapInput{Pixels: tc.pixels, Width: 2, Height: 1, Stride: tc.stride, Format: tc.format})
			if err != nil {
				t.Fatalf("EncodeRDPGFXUncompressed: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("payload = %x, want %x", got, want)
			}
		})
	}
}

func TestEncodeRDPGFXUncompressedStride(t *testing.T) {
	pixels := []byte{
		0x10, 0x20, 0x30, 0xff, 0x40, 0x50, 0x60, 0xff, 0xee, 0xee, 0xee, 0xee,
		0x70, 0x80, 0x90, 0xff, 0xa0, 0xb0, 0xc0, 0xff, 0xdd, 0xdd, 0xdd, 0xdd,
	}
	got, err := EncodeRDPGFXUncompressed(BitmapInput{Pixels: pixels, Width: 2, Height: 2, Stride: 12, Format: PixelFormatRGBA})
	if err != nil {
		t.Fatalf("EncodeRDPGFXUncompressed: %v", err)
	}
	want := []byte{
		0x30, 0x20, 0x10, 0xff, 0x60, 0x50, 0x40, 0xff,
		0x90, 0x80, 0x70, 0xff, 0xc0, 0xb0, 0xa0, 0xff,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("payload = %x, want %x", got, want)
	}
}

func TestEncodeRDPGFXUncompressedRejectsInvalidInput(t *testing.T) {
	valid := []byte{0, 0, 0, 0}
	cases := []BitmapInput{
		{},
		{Pixels: valid, Width: 1, Height: 1, Stride: 3, Format: PixelFormatRGBA},
		{Pixels: []byte{0, 0, 0}, Width: 1, Height: 1, Stride: 4, Format: PixelFormatRGBA},
		{Pixels: valid, Width: 1, Height: 1, Stride: 4, Format: 99},
	}
	for _, tc := range cases {
		if _, err := EncodeRDPGFXUncompressed(tc); err == nil {
			t.Fatalf("EncodeRDPGFXUncompressed(%#v) err = nil", tc)
		}
	}
}

func TestRDPGFXUncompressedWireToSurfaceDeterministicBytes(t *testing.T) {
	payload, err := EncodeRDPGFXUncompressed(BitmapInput{Pixels: []byte{0x10, 0x20, 0x30, 0xff}, Width: 1, Height: 1, Stride: 4, Format: PixelFormatRGBA})
	if err != nil {
		t.Fatalf("EncodeRDPGFXUncompressed: %v", err)
	}
	wire, err := BuildRDPGFXWireToSurface1(3, RDPGFXCodecUncompressed, RDPGFXPixelFormatXRGB8888, Rect{Right: 1, Bottom: 1}, payload)
	if err != nil {
		t.Fatalf("BuildRDPGFXWireToSurface1: %v", err)
	}
	if binary.LittleEndian.Uint16(wire[0:2]) != RDPGFXCmdWireToSurface1 {
		t.Fatalf("cmd = 0x%04x", binary.LittleEndian.Uint16(wire[0:2]))
	}
	if binary.LittleEndian.Uint16(wire[10:12]) != RDPGFXCodecUncompressed {
		t.Fatalf("codec = 0x%04x", binary.LittleEndian.Uint16(wire[10:12]))
	}
	wantSuffix := []byte{0x04, 0x00, 0x00, 0x00, 0x30, 0x20, 0x10, 0xff}
	if !bytes.Equal(wire[len(wire)-len(wantSuffix):], wantSuffix) {
		t.Fatalf("wire suffix = %x, want %x", wire[len(wire)-len(wantSuffix):], wantSuffix)
	}
}
