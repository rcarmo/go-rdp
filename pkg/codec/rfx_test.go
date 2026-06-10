package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestPublicRFXQuantAndRGBConversion(t *testing.T) {
	q, err := ParseRFXQuant([]byte{0x12, 0x34, 0x56, 0x78, 0x9a})
	if err != nil {
		t.Fatalf("ParseRFXQuant: %v", err)
	}
	if q.Values != ([10]uint8{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}) {
		t.Fatalf("quant = %v", q.Values)
	}
	if _, err := ParseRFXQuant([]byte{0x12}); err == nil {
		t.Fatal("ParseRFXQuant accepted malformed payload")
	}
	y, co, cg := RFXRGBToYCoCg(100, 50, 10)
	if y != 52 || co != 90 || cg != -5 {
		t.Fatalf("RFXRGBToYCoCg = %d/%d/%d, want 52/90/-5", y, co, cg)
	}
	if got := DefaultRFXQuantBytes(); !bytes.Equal(got, []byte{0x66, 0x66, 0x77, 0x88, 0x89}) {
		t.Fatalf("DefaultRFXQuantBytes = %x", got)
	}
}

func TestPublicRFXComponentWrappers(t *testing.T) {
	coeff := make([]int16, RFXTilePixels)
	for i := range coeff {
		coeff[i] = 64
	}
	if err := ForwardRFXDWT53(coeff); err != nil {
		t.Fatalf("ForwardRFXDWT53: %v", err)
	}
	if err := QuantizeRFXComponent(coeff, DefaultRFXQuant()); err != nil {
		t.Fatalf("QuantizeRFXComponent: %v", err)
	}
	packed, err := SerializeRFXComponentForRLGR(coeff)
	if err != nil {
		t.Fatalf("SerializeRFXComponentForRLGR: %v", err)
	}
	encoded, err := EncodeRFXRLGR(packed, 1)
	if err != nil || len(encoded) == 0 {
		t.Fatalf("EncodeRFXRLGR len=%d err=%v", len(encoded), err)
	}
	if err := ForwardRFXDWT53(coeff[:10]); err == nil {
		t.Fatal("ForwardRFXDWT53 accepted short buffer")
	}
	if _, err := EncodeRFXRLGR(packed, 2); err == nil {
		t.Fatal("EncodeRFXRLGR accepted invalid mode")
	}
}

func TestPublicEncodeRFXSingleTileFrameAndParse(t *testing.T) {
	input := gradientRFXInput()
	msg, err := EncodeRFXSingleTileFrame(input, RFXTileSize, RFXTileSize, 11, 0, 0, nil)
	if err != nil {
		t.Fatalf("EncodeRFXSingleTileFrame: %v", err)
	}
	again, err := EncodeRFXSingleTileFrame(input, RFXTileSize, RFXTileSize, 11, 0, 0, nil)
	if err != nil || !bytes.Equal(msg, again) {
		t.Fatalf("EncodeRFXSingleTileFrame deterministic err=%v", err)
	}
	var types []uint16
	for off := 0; off+6 <= len(msg); {
		bt := binary.LittleEndian.Uint16(msg[off : off+2])
		bl := int(binary.LittleEndian.Uint32(msg[off+2 : off+6]))
		if bl < 6 || off+bl > len(msg) {
			t.Fatalf("invalid block len=%d off=%d", bl, off)
		}
		types = append(types, bt)
		off += bl
	}
	want := []uint16{rfxBlockTypeSync, rfxBlockTypeCodecVersions, rfxBlockTypeChannels, rfxBlockTypeContext, rfxBlockTypeFrameBegin, rfxBlockTypeRegion, rfxBlockTypeTileset, rfxBlockTypeFrameEnd}
	if len(types) != len(want) {
		t.Fatalf("types=%x want=%x", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("type[%d]=0x%04x want 0x%04x", i, types[i], want[i])
		}
	}
	if _, err := EncodeRFXSingleTileFrame(input, 0, RFXTileSize, 1, 0, 0, nil); err == nil {
		t.Fatal("EncodeRFXSingleTileFrame accepted zero width")
	}
}

func TestPublicRFXTileAndSurfaceBits(t *testing.T) {
	input := gradientRFXInput()
	y, co, cg, err := EncodeRFXTileComponents(input, 0, 0, nil)
	if err != nil || len(y) == 0 || len(co) == 0 || len(cg) == 0 {
		t.Fatalf("EncodeRFXTileComponents lens=%d/%d/%d err=%v", len(y), len(co), len(cg), err)
	}
	tile, err := BuildRFXTileBlock(0, 0, y, co, cg)
	if err != nil || binary.LittleEndian.Uint16(tile[0:2]) != rfxBlockTypeTile {
		t.Fatalf("BuildRFXTileBlock len=%d err=%v", len(tile), err)
	}
	cmd, err := BuildRFXSetSurfaceBits(Rect{Right: RFXTileSize - 1, Bottom: RFXTileSize - 1}, 4, input, 1, nil)
	if err != nil {
		t.Fatalf("BuildRFXSetSurfaceBits: %v", err)
	}
	if binary.LittleEndian.Uint16(cmd[0:2]) != SurfaceCmdSetSurfaceBits || cmd[13] != 4 {
		t.Fatalf("unexpected SurfaceBits header: %x", cmd[:22])
	}
	if bitmapLen := int(binary.LittleEndian.Uint32(cmd[18:22])); bitmapLen != len(cmd)-22 || bitmapLen == 0 {
		t.Fatalf("bitmapLen=%d total=%d", bitmapLen, len(cmd))
	}
}

func BenchmarkEncodeRFXSingleTileFrame(b *testing.B) {
	input := gradientRFXInput()
	b.ReportAllocs()
	b.SetBytes(int64(RFXTileSize * RFXTileSize * 4))
	for i := 0; i < b.N; i++ {
		msg, err := EncodeRFXSingleTileFrame(input, RFXTileSize, RFXTileSize, uint32(i+1), 0, 0, nil)
		if err != nil || len(msg) == 0 {
			b.Fatalf("EncodeRFXSingleTileFrame len=%d err=%v", len(msg), err)
		}
	}
}

func gradientRFXInput() BitmapInput {
	pixels := make([]byte, RFXTileSize*RFXTileSize*4)
	for y := 0; y < RFXTileSize; y++ {
		for x := 0; x < RFXTileSize; x++ {
			i := (y*RFXTileSize + x) * 4
			pixels[i+0] = byte(x)
			pixels[i+1] = byte(y)
			pixels[i+2] = byte((x + y) & 0xff)
			pixels[i+3] = 0xff
		}
	}
	return BitmapInput{Pixels: pixels, Width: RFXTileSize, Height: RFXTileSize, Stride: RFXTileSize * 4, Format: PixelFormatRGBA}
}
