package codec

import (
	"bytes"
	"testing"
)

func TestEncodePlanarNoAlphaRoundTripRGBAAndBGRA(t *testing.T) {
	rgba := []byte{
		0x10, 0x20, 0x30, 0xff, 0x40, 0x50, 0x60, 0xff,
		0x70, 0x80, 0x90, 0xff, 0xa0, 0xb0, 0xc0, 0xff,
	}
	bgra := []byte{
		0x30, 0x20, 0x10, 0xff, 0x60, 0x50, 0x40, 0xff,
		0x90, 0x80, 0x70, 0xff, 0xc0, 0xb0, 0xa0, 0xff,
	}
	wantDecodedRGBA := []byte{
		0x70, 0x80, 0x90, 0xff, 0xa0, 0xb0, 0xc0, 0xff,
		0x10, 0x20, 0x30, 0xff, 0x40, 0x50, 0x60, 0xff,
	}
	for _, tc := range []struct {
		name   string
		data   []byte
		format PixelFormat
	}{
		{"rgba", rgba, PixelFormatRGBA},
		{"bgra", bgra, PixelFormatBGRA},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := EncodePlanarNoAlpha(PlanarInput{Pixels: tc.data, Width: 2, Height: 2, Stride: 8, Format: tc.format})
			if err != nil {
				t.Fatalf("EncodePlanarNoAlpha: %v", err)
			}
			if len(payload) == 0 || payload[0] != PlanarFlagRLE|PlanarFlagNoAlpha {
				t.Fatalf("unexpected payload header: %x", payload)
			}
			decoded := DecodePlanar(payload, 2, 2)
			if !bytes.Equal(decoded, wantDecodedRGBA) {
				t.Fatalf("decoded = %x, want %x", decoded, wantDecodedRGBA)
			}
		})
	}
}

func TestEncodePlanarNoAlphaDeterministicPayload(t *testing.T) {
	data := []byte{
		0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0x00, 0xff,
		0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0x00, 0xff,
	}
	payload, err := EncodePlanarNoAlpha(PlanarInput{Pixels: data, Width: 2, Height: 2, Stride: 8, Format: PixelFormatRGBA})
	if err != nil {
		t.Fatalf("EncodePlanarNoAlpha: %v", err)
	}
	want := []byte{0x30, 0x20, 0x00, 0x00, 0x20, 0x00, 0x00, 0x20, 0x00, 0x00, 0x20, 0x00, 0x00, 0x20, 0x00, 0x00, 0x20, 0x00, 0x00}
	if !bytes.Equal(payload, want) {
		t.Fatalf("payload = %x, want %x", payload, want)
	}
}

func TestPlanarDeltaByteWrapBehavior(t *testing.T) {
	cases := map[int]byte{
		0:    0x00,
		1:    0x02,
		-1:   0x01,
		127:  0xfe,
		-128: 0xff,
		128:  0xff, // wraps to -128
		-129: 0xfe, // wraps to 127
	}
	for delta, want := range cases {
		if got := PlanarDeltaByte(delta); got != want {
			t.Fatalf("PlanarDeltaByte(%d) = 0x%02x, want 0x%02x", delta, got, want)
		}
	}
}

func TestEncodePlanarNoAlphaRejectsInvalidInput(t *testing.T) {
	valid := []byte{0, 0, 0, 0}
	cases := []PlanarInput{
		{},
		{Pixels: valid, Width: 1, Height: 1, Stride: 3, Format: PixelFormatRGBA},
		{Pixels: []byte{0, 0, 0}, Width: 1, Height: 1, Stride: 4, Format: PixelFormatRGBA},
		{Pixels: valid, Width: 1, Height: 1, Stride: 4, Format: 99},
	}
	for _, tc := range cases {
		if _, err := EncodePlanarNoAlpha(tc); err == nil {
			t.Fatalf("EncodePlanarNoAlpha(%#v) err = nil", tc)
		}
	}
}

func BenchmarkEncodePlanarNoAlpha320x240(b *testing.B) {
	width, height := 320, 240
	pixels := make([]byte, width*height*4)
	for i := 0; i < len(pixels); i += 4 {
		pixels[i] = byte(i)
		pixels[i+1] = byte(i >> 8)
		pixels[i+2] = byte(i >> 16)
		pixels[i+3] = 0xff
	}
	input := PlanarInput{Pixels: pixels, Width: width, Height: height, Stride: width * 4, Format: PixelFormatRGBA}
	b.ReportAllocs()
	b.SetBytes(int64(width * height * 4))
	for i := 0; i < b.N; i++ {
		payload, err := EncodePlanarNoAlpha(input)
		if err != nil || len(payload) == 0 {
			b.Fatalf("EncodePlanarNoAlpha len=%d err=%v", len(payload), err)
		}
	}
}
