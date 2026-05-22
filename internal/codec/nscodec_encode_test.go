package codec

import "testing"

func TestEncodeNSCodecRawBGRA(t *testing.T) {
	pixels := []byte{
		0x00, 0x00, 0x00, 0xff,
		0xff, 0xff, 0xff, 0xff,
		0x44, 0x33, 0x22, 0xff,
		0x10, 0x80, 0xf0, 0xff,
	}
	encoded, ok := EncodeNSCodecRawBGRA(pixels, 2, 2, 8)
	if !ok {
		t.Fatal("EncodeNSCodecRawBGRA ok = false")
	}
	if len(encoded) != 20+2*2*3 {
		t.Fatalf("encoded len = %d", len(encoded))
	}
	decoded, err := Decode(encoded, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(pixels) {
		t.Fatalf("decoded len = %d", len(decoded))
	}
	for i := 0; i < 4; i++ {
		wantR := pixels[i*4+2]
		wantG := pixels[i*4+1]
		wantB := pixels[i*4+0]
		gotR := decoded[i*4+0]
		gotG := decoded[i*4+1]
		gotB := decoded[i*4+2]
		if absByteDiff(gotR, wantR) > 2 || absByteDiff(gotG, wantG) > 2 || absByteDiff(gotB, wantB) > 2 {
			t.Fatalf("pixel %d decoded rgb=(%d,%d,%d), want near (%d,%d,%d)", i, gotR, gotG, gotB, wantR, wantG, wantB)
		}
	}
}

func TestEncodeNSCodecRawRejectsInvalid(t *testing.T) {
	if _, ok := EncodeNSCodecRawBGRA(nil, 1, 1, 4); ok {
		t.Fatal("expected short input to be rejected")
	}
	if _, ok := EncodeNSCodecRawBGRA(make([]byte, 4), 1, 1, 3); ok {
		t.Fatal("expected short stride to be rejected")
	}
	if _, ok := EncodeNSCodecRawBGRA(make([]byte, 4), 0, 1, 4); ok {
		t.Fatal("expected invalid dimensions to be rejected")
	}
}

func absByteDiff(a, b byte) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}
