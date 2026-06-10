package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestEncodeClearCodecMinimalSolidRect(t *testing.T) {
	payload, err := EncodeClearCodecMinimal(solidClearInput(8, 8, PixelFormatRGBA, 0x11, 0x22, 0x33))
	if err != nil {
		t.Fatalf("EncodeClearCodecMinimal: %v", err)
	}
	if len(payload) != 22 || string(payload[:4]) != ClearCodecMagic {
		t.Fatalf("payload len=%d magic=%q", len(payload), payload[:4])
	}
	if got := binary.LittleEndian.Uint16(payload[8:10]); got != 1 {
		t.Fatalf("rect count=%d", got)
	}
	if payload[10] != ClearCodecOpSolidRect || payload[19] != 0x11 || payload[20] != 0x22 || payload[21] != 0x33 {
		t.Fatalf("solid rect payload=%x", payload)
	}
}

func TestEncodeClearCodecMinimalRawRectRGB565(t *testing.T) {
	pixels := make([]byte, 8*8*4)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			i := (y*8 + x) * 4
			pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = byte(x*31), byte(y*31), byte((x+y)*15), 0xff
		}
	}
	payload, err := EncodeClearCodecMinimal(BitmapInput{Pixels: pixels, Width: 8, Height: 8, Stride: 32, Format: PixelFormatRGBA})
	if err != nil {
		t.Fatalf("EncodeClearCodecMinimal: %v", err)
	}
	rects, ok := parseClearCodecRectHeaders(payload)
	if !ok || len(rects) != 1 {
		t.Fatalf("rects len=%d ok=%t", len(rects), ok)
	}
	if payload[10] != ClearCodecOpRawRect {
		t.Fatalf("op=%d want raw", payload[10])
	}
	dataLen := int(binary.LittleEndian.Uint32(payload[19:23]))
	if dataLen != 8*8*2 {
		t.Fatalf("dataLen=%d", dataLen)
	}
	first565 := binary.LittleEndian.Uint16(payload[23:25])
	if want := RGB565FromRGB(0, 0, 0); first565 != want {
		t.Fatalf("first RGB565=0x%04x want 0x%04x", first565, want)
	}
}

func TestRGB565FromRGB(t *testing.T) {
	cases := []struct {
		r, g, b byte
		want    uint16
	}{
		{0xff, 0xff, 0xff, 0xffff},
		{0xff, 0x00, 0x00, 0xf800},
		{0x00, 0xff, 0x00, 0x07e0},
		{0x00, 0x00, 0xff, 0x001f},
	}
	for _, tc := range cases {
		if got := RGB565FromRGB(tc.r, tc.g, tc.b); got != tc.want {
			t.Fatalf("RGB565FromRGB(%02x,%02x,%02x)=0x%04x want 0x%04x", tc.r, tc.g, tc.b, got, tc.want)
		}
	}
}

func TestEncodeClearCodecMinimalUsesSolidBandsAndTiles(t *testing.T) {
	input := solidClearInput(128, 64, PixelFormatRGBA, 0x11, 0x22, 0x33)
	// Make right tile non-solid relative to left tile but internally solid.
	for y := 0; y < input.Height; y++ {
		for x := 64; x < input.Width; x++ {
			i := y*input.Stride + x*4
			input.Pixels[i], input.Pixels[i+1], input.Pixels[i+2] = 0xaa, 0xbb, 0xcc
		}
	}
	payload, err := EncodeClearCodecMinimal(input)
	if err != nil {
		t.Fatalf("EncodeClearCodecMinimal: %v", err)
	}
	rects, ok := parseClearCodecRectHeaders(payload)
	if !ok || len(rects) != 2 {
		t.Fatalf("rects len=%d ok=%t", len(rects), ok)
	}
	if rects[0] != ([4]uint16{0, 0, 64, 64}) || rects[1] != ([4]uint16{64, 0, 64, 64}) {
		t.Fatalf("rects=%v", rects)
	}
}

func TestEncodeClearCodecMinimalLargeFrameSplitsRects(t *testing.T) {
	payload, err := EncodeClearCodecMinimal(solidClearInput(1024, 256, PixelFormatRGBA, 0x11, 0x22, 0x33))
	if err != nil {
		t.Fatalf("EncodeClearCodecMinimal: %v", err)
	}
	rects, ok := parseClearCodecRectHeaders(payload)
	if !ok || len(rects) != 1 {
		t.Fatalf("whole-frame solid should use one rect, got len=%d ok=%t", len(rects), ok)
	}
	input := checkerClearInput(128, 128)
	payload, err = EncodeClearCodecMinimal(input)
	if err != nil {
		t.Fatalf("checker EncodeClearCodecMinimal: %v", err)
	}
	rects, ok = parseClearCodecRectHeaders(payload)
	if !ok || len(rects) != 4 {
		t.Fatalf("checker rects len=%d ok=%t", len(rects), ok)
	}
}

func TestEncodeClearCodecMinimalDeterministic(t *testing.T) {
	input := checkerClearInput(64, 64)
	first, err := EncodeClearCodecMinimal(input)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := EncodeClearCodecMinimal(input)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("ClearCodec output is not deterministic")
	}
}

func TestEncodeClearCodecMinimalRejectsInvalidAndExpansion(t *testing.T) {
	if _, err := EncodeClearCodecMinimal(BitmapInput{}); err == nil {
		t.Fatal("accepted empty input")
	}
	if _, err := EncodeClearCodecMinimal(BitmapInput{Pixels: []byte{1, 2, 3}, Width: 1, Height: 1, Stride: 4, Format: PixelFormatRGBA}); err == nil {
		t.Fatal("accepted short input")
	}
	if _, err := EncodeClearCodecMinimal(BitmapInput{Pixels: []byte{1, 2, 3, 4}, Width: 1, Height: 1, Stride: 4, Format: PixelFormatBGR}); err == nil {
		t.Fatal("accepted unsupported format")
	}
	// A 1x1 solid ClearCodec envelope is larger than raw 32-bpp input, so it must be rejected.
	if _, err := EncodeClearCodecMinimal(solidClearInput(1, 1, PixelFormatRGBA, 1, 2, 3)); err == nil {
		t.Fatal("accepted expanding 1x1 payload")
	}
}

func TestBuildClearCodecWireToSurface(t *testing.T) {
	input := solidClearInput(8, 8, PixelFormatRGBA, 0x11, 0x22, 0x33)
	wire, err := BuildClearCodecWireToSurface(2, RDPGFXPixelFormatXRGB8888, Rect{Right: 8, Bottom: 8}, input)
	if err != nil {
		t.Fatalf("BuildClearCodecWireToSurface: %v", err)
	}
	if binary.LittleEndian.Uint16(wire[0:2]) != RDPGFXCmdWireToSurface1 {
		t.Fatalf("cmd=0x%04x", binary.LittleEndian.Uint16(wire[0:2]))
	}
	if binary.LittleEndian.Uint16(wire[10:12]) != RDPGFXCodecClearCodec {
		t.Fatalf("codec=0x%04x", binary.LittleEndian.Uint16(wire[10:12]))
	}
	if !bytes.Contains(wire, []byte(ClearCodecMagic)) {
		t.Fatalf("wire missing ClearCodec magic: %x", wire)
	}
}

func BenchmarkEncodeClearCodecMinimalSolid(b *testing.B) {
	input := solidClearInput(320, 240, PixelFormatRGBA, 0x11, 0x22, 0x33)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		payload, err := EncodeClearCodecMinimal(input)
		if err != nil || len(payload) == 0 {
			b.Fatalf("payload len=%d err=%v", len(payload), err)
		}
	}
}

func parseClearCodecRectHeaders(payload []byte) ([][4]uint16, bool) {
	if len(payload) < 10 || string(payload[:4]) != ClearCodecMagic {
		return nil, false
	}
	rectCount := int(binary.LittleEndian.Uint16(payload[8:10]))
	if rectCount == 1 && len(payload) == 22 && payload[10] == ClearCodecOpSolidRect {
		return [][4]uint16{{
			binary.LittleEndian.Uint16(payload[11:13]),
			binary.LittleEndian.Uint16(payload[13:15]),
			binary.LittleEndian.Uint16(payload[15:17]),
			binary.LittleEndian.Uint16(payload[17:19]),
		}}, true
	}
	rects := make([][4]uint16, 0, rectCount)
	off := 10
	for i := 0; i < rectCount; i++ {
		if off+13 > len(payload) {
			return nil, false
		}
		op := payload[off]
		x := binary.LittleEndian.Uint16(payload[off+1 : off+3])
		y := binary.LittleEndian.Uint16(payload[off+3 : off+5])
		w := binary.LittleEndian.Uint16(payload[off+5 : off+7])
		h := binary.LittleEndian.Uint16(payload[off+7 : off+9])
		dataLen := int(binary.LittleEndian.Uint32(payload[off+9 : off+13]))
		rects = append(rects, [4]uint16{x, y, w, h})
		off += 13
		switch op {
		case ClearCodecOpSolidRect:
			if dataLen != 0 || off+3 > len(payload) {
				return nil, false
			}
			off += 3
		case ClearCodecOpRawRect:
			if dataLen <= 0 || off+dataLen > len(payload) {
				return nil, false
			}
			off += dataLen
		default:
			return nil, false
		}
	}
	return rects, off == len(payload)
}

func solidClearInput(width, height int, format PixelFormat, r, g, b byte) BitmapInput {
	pixels := make([]byte, width*height*4)
	for i := 0; i < len(pixels); i += 4 {
		if format == PixelFormatBGRA {
			pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = b, g, r, 0xff
		} else {
			pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = r, g, b, 0xff
		}
	}
	return BitmapInput{Pixels: pixels, Width: width, Height: height, Stride: width * 4, Format: format}
}

func checkerClearInput(width, height int) BitmapInput {
	pixels := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 4
			pixels[i] = byte((x * 7) & 0xff)
			pixels[i+1] = byte((y * 5) & 0xff)
			pixels[i+2] = byte(((x + y) * 3) & 0xff)
			pixels[i+3] = 0xff
		}
	}
	return BitmapInput{Pixels: pixels, Width: width, Height: height, Stride: width * 4, Format: PixelFormatRGBA}
}
