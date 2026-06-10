package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBuildBitmapUpdateDeterministicBytes(t *testing.T) {
	rect := BitmapUpdateRect{Left: 1, Top: 2, Right: 2, Bottom: 3, Width: 2, Height: 2, BitsPerPixel: BitmapBPP24, Data: []byte{1, 2, 3, 4}}
	got, err := BuildBitmapUpdate([]BitmapUpdateRect{rect})
	if err != nil {
		t.Fatalf("BuildBitmapUpdate: %v", err)
	}
	want := []byte{
		0x01, 0x00, // updateType BITMAP
		0x01, 0x00, // rectangle count
		0x01, 0x00, 0x02, 0x00, 0x02, 0x00, 0x03, 0x00,
		0x02, 0x00, 0x02, 0x00,
		0x18, 0x00,
		0x00, 0x00,
		0x04, 0x00,
		0x01, 0x02, 0x03, 0x04,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("BuildBitmapUpdate = %x, want %x", got, want)
	}
}

func TestBuildBitmapUpdateRejectsMalformedGeometry(t *testing.T) {
	valid := BitmapUpdateRect{Left: 0, Top: 0, Right: 0, Bottom: 0, Width: 1, Height: 1, BitsPerPixel: BitmapBPP24, Data: []byte{1, 2, 3, 0}}
	cases := []BitmapUpdateRect{
		{},
		{Left: 2, Top: 0, Right: 1, Bottom: 0, Width: 1, Height: 1, BitsPerPixel: BitmapBPP24, Data: []byte{1}},
		{Left: 0, Top: 0, Right: 1, Bottom: 0, Width: 1, Height: 1, BitsPerPixel: BitmapBPP24, Data: []byte{1}},
		{Left: 0, Top: 0, Right: 0, Bottom: 0, Width: 1, Height: 1, BitsPerPixel: 32, Data: []byte{1}},
		{Left: 0, Top: 0, Right: 0, Bottom: 0, Width: 1, Height: 1, BitsPerPixel: BitmapBPP24},
	}
	for _, tc := range cases {
		if _, err := BuildBitmapUpdate([]BitmapUpdateRect{tc}); err == nil {
			t.Fatalf("BuildBitmapUpdate(%#v) err = nil", tc)
		}
	}
	if _, err := BuildBitmapUpdate([]BitmapUpdateRect{valid}); err != nil {
		t.Fatalf("valid BuildBitmapUpdate: %v", err)
	}
}

func TestEncodeBitmapRLECopyRoundTripsSupportedBPP(t *testing.T) {
	cases := []struct {
		name          string
		bpp           uint16
		bytesPerPixel int
		decode        func(src, dst []byte, rowDelta int) bool
	}{
		{"8bpp", BitmapBPP8, 1, RLEDecompress8},
		{"15bpp", BitmapBPP15, 2, RLEDecompress15},
		{"16bpp", BitmapBPP16, 2, RLEDecompress16},
		{"24bpp", BitmapBPP24, 3, RLEDecompress24},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			width, height := 4, 2
			rowBytes, ok := AlignedBitmapRowBytes(width, tc.bpp)
			if !ok {
				t.Fatal("AlignedBitmapRowBytes ok=false")
			}
			visible := width * tc.bytesPerPixel
			input := make([]byte, rowBytes*height)
			for i := 0; i < visible; i++ {
				input[i] = byte(i + 1)
				input[rowBytes+i] = byte(i + 1 + visible)
			}
			encoded, err := EncodeBitmapRLECopy(input, width, height, tc.bpp)
			if err != nil {
				t.Fatalf("EncodeBitmapRLECopy: %v", err)
			}
			decoded := make([]byte, rowBytes*height)
			if !tc.decode(encoded, decoded, rowBytes) {
				t.Fatalf("decode failed for %x", encoded)
			}
			// Existing go-rdp decoders output decompressed rows in bitmap wire order
			// (bottom row first), matching the RLE stream emitted by the server encoder.
			want := make([]byte, rowBytes*height)
			copy(want[0:visible], input[rowBytes:rowBytes+visible])
			copy(want[rowBytes:rowBytes+visible], input[0:visible])
			if !bytes.Equal(decoded, want) {
				t.Fatalf("decoded = %x, want %x", decoded, want)
			}
		})
	}
}

func TestBuildCompressedBitmapRLEUpdate(t *testing.T) {
	rowBytes, _ := AlignedBitmapRowBytes(80, BitmapBPP24)
	rect := BitmapUpdateRect{Left: 0, Top: 0, Right: 79, Bottom: 0, Width: 80, Height: 1, BitsPerPixel: BitmapBPP24, Data: make([]byte, rowBytes)}
	update, err := BuildCompressedBitmapRLEUpdate([]BitmapUpdateRect{rect})
	if err != nil {
		t.Fatalf("BuildCompressedBitmapRLEUpdate: %v", err)
	}
	if binary.LittleEndian.Uint16(update[0:2]) != UpdateTypeBitmap || binary.LittleEndian.Uint16(update[2:4]) != 1 {
		t.Fatalf("unexpected bitmap update header %x", update[:4])
	}
	flags := binary.LittleEndian.Uint16(update[4+14 : 4+16])
	if flags != BitmapCompressionFlag|NoBitmapCompressionHeader {
		t.Fatalf("flags = 0x%04x", flags)
	}
	dataLen := int(binary.LittleEndian.Uint16(update[4+16 : 4+18]))
	if dataLen <= 0 || dataLen >= len(rect.Data) {
		t.Fatalf("dataLen = %d raw=%d", dataLen, len(rect.Data))
	}
}

func TestBuildCompressedBitmapRLEUpdateRejectsExpansion(t *testing.T) {
	rowBytes, _ := AlignedBitmapRowBytes(80, BitmapBPP24)
	rect := BitmapUpdateRect{Left: 0, Top: 0, Right: 79, Bottom: 0, Width: 80, Height: 1, BitsPerPixel: BitmapBPP24, Data: make([]byte, rowBytes)}
	for i := range rect.Data {
		rect.Data[i] = byte(i)
	}
	if _, err := BuildCompressedBitmapRLEUpdate([]BitmapUpdateRect{rect}); err == nil {
		t.Fatal("BuildCompressedBitmapRLEUpdate accepted expanding payload")
	}
}

func TestEncodeBitmapRLECopyRejectsInvalid(t *testing.T) {
	if _, err := EncodeBitmapRLECopy([]byte{0, 0, 0, 0}, 1, 1, 32); err == nil {
		t.Fatal("EncodeBitmapRLECopy accepted 32bpp")
	}
	if _, err := EncodeBitmapRLECopy([]byte{0}, 2, 2, BitmapBPP24); err == nil {
		t.Fatal("EncodeBitmapRLECopy accepted short data")
	}
	if _, err := EncodeBitmapRLECopy(nil, 0, 1, BitmapBPP24); err == nil {
		t.Fatal("EncodeBitmapRLECopy accepted zero width")
	}
}

func TestAlignedBitmapRowBytes(t *testing.T) {
	if got, ok := AlignedBitmapRowBytes(1, BitmapBPP24); !ok || got != 4 {
		t.Fatalf("AlignedBitmapRowBytes(1,24) = %d,%t, want 4,true", got, ok)
	}
	if got, ok := AlignedBitmapRowBytes(80, BitmapBPP24); !ok || got != 240 {
		t.Fatalf("AlignedBitmapRowBytes(80,24) = %d,%t, want 240,true", got, ok)
	}
	if _, ok := AlignedBitmapRowBytes(1, 32); ok {
		t.Fatal("AlignedBitmapRowBytes accepted 32bpp")
	}
}
