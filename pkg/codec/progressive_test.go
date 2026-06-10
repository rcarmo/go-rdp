package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestMarshalParseProgressivePayload(t *testing.T) {
	in := progressiveSeed()
	wire, err := MarshalProgressivePayload(in)
	if err != nil {
		t.Fatalf("MarshalProgressivePayload: %v", err)
	}
	want := []byte{
		0x40, 0x00, 0x40, 0x00, 0x02, 0x06, 0x01, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x40, 0x00, 0x40, 0x00,
		0x05, 0x00, 0x00, 0x00, 1, 2, 3, 4, 5,
	}
	if !bytes.Equal(wire, want) {
		t.Fatalf("wire=%x want=%x", wire, want)
	}
	decoded, err := ParseProgressivePayload(wire)
	if err != nil {
		t.Fatalf("ParseProgressivePayload: %v", err)
	}
	if decoded.Width != in.Width || decoded.Height != in.Height || decoded.LayerCount != in.LayerCount || decoded.Quant != in.Quant || len(decoded.RegionRects) != 1 || !bytes.Equal(decoded.EncodedData, in.EncodedData) {
		t.Fatalf("decoded=%#v", decoded)
	}
}

func TestProgressivePayloadAliasAndCopyBehavior(t *testing.T) {
	wire, err := MarshalProgressivePayload(progressiveSeed())
	if err != nil {
		t.Fatal(err)
	}
	alias, err := ParseProgressivePayloadAlias(wire)
	if err != nil {
		t.Fatalf("ParseProgressivePayloadAlias: %v", err)
	}
	copied, err := ParseProgressivePayload(wire)
	if err != nil {
		t.Fatalf("ParseProgressivePayload: %v", err)
	}
	alias.EncodedData[0] ^= 0xff
	if wire[len(wire)-5] != (byte(1) ^ 0xff) {
		t.Fatalf("alias did not modify source wire")
	}
	if copied.EncodedData[0] != 1 {
		t.Fatalf("copied data was aliased")
	}
}

func TestProgressivePayloadRejectsInvalid(t *testing.T) {
	valid := progressiveSeed()
	cases := []ProgressivePayload{
		{},
		{Width: 64, Height: 64, LayerCount: 0, RegionRects: valid.RegionRects, EncodedData: valid.EncodedData},
		{Width: 64, Height: 64, LayerCount: 9, RegionRects: valid.RegionRects, EncodedData: valid.EncodedData},
		{Width: 64, Height: 64, LayerCount: 1, RegionRects: []ProgressiveRect{{Left: 1, Right: 1, Bottom: 1}}, EncodedData: []byte{1}},
		{Width: 64, Height: 64, LayerCount: 1, RegionRects: []ProgressiveRect{{Right: 65, Bottom: 1}}, EncodedData: []byte{1}},
		{Width: 64, Height: 64, LayerCount: 1, RegionRects: valid.RegionRects},
	}
	for _, tc := range cases {
		if _, err := MarshalProgressivePayload(tc); err == nil {
			t.Fatalf("MarshalProgressivePayload(%#v) err=nil", tc)
		}
	}
}

func TestParseProgressivePayloadRejectsMalformed(t *testing.T) {
	if _, err := ParseProgressivePayload([]byte{1, 2, 3}); err == nil {
		t.Fatal("accepted short payload")
	}
	wire, err := MarshalProgressivePayload(progressiveSeed())
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(wire); i++ {
		if _, err := ParseProgressivePayload(wire[:i]); err == nil {
			t.Fatalf("accepted truncated payload len=%d", i)
		}
	}
	bad := append([]byte(nil), wire...)
	binary.LittleEndian.PutUint16(bad[6:8], ProgressiveMaxRects+1)
	if _, err := ParseProgressivePayload(bad); err == nil {
		t.Fatal("accepted oversized rect count")
	}
	bad = append([]byte(nil), wire...)
	binary.LittleEndian.PutUint32(bad[16:20], uint32(len(wire)))
	if _, err := ParseProgressivePayload(bad); err == nil {
		t.Fatal("accepted oversized data length")
	}
}

func TestBuildProgressiveWireToSurface(t *testing.T) {
	wire, err := BuildProgressiveWireToSurface(7, RDPGFXPixelFormatXRGB8888, Rect{Right: 64, Bottom: 64}, progressiveSeed(), false)
	if err != nil {
		t.Fatalf("BuildProgressiveWireToSurface: %v", err)
	}
	if binary.LittleEndian.Uint16(wire[10:12]) != RDPGFXCodecCAProgressive {
		t.Fatalf("codec=0x%04x", binary.LittleEndian.Uint16(wire[10:12]))
	}
	wire, err = BuildProgressiveWireToSurface(7, RDPGFXPixelFormatXRGB8888, Rect{Right: 64, Bottom: 64}, progressiveSeed(), true)
	if err != nil {
		t.Fatalf("BuildProgressiveWireToSurface v2: %v", err)
	}
	if binary.LittleEndian.Uint16(wire[10:12]) != RDPGFXCodecCAProgressiveV2 {
		t.Fatalf("v2 codec=0x%04x", binary.LittleEndian.Uint16(wire[10:12]))
	}
}

func FuzzParseProgressivePayload(f *testing.F) {
	wire, err := MarshalProgressivePayload(progressiveSeed())
	if err == nil {
		f.Add(wire)
	}
	f.Add([]byte{1, 2, 3})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseProgressivePayload(data)
	})
}

func BenchmarkParseProgressivePayload(b *testing.B) {
	wire, err := MarshalProgressivePayload(progressiveSeed())
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		parsed, err := ParseProgressivePayload(wire)
		if err != nil || len(parsed.EncodedData) == 0 {
			b.Fatalf("parse err=%v", err)
		}
	}
}

func progressiveSeed() ProgressivePayload {
	return ProgressivePayload{
		Width:      64,
		Height:     64,
		LayerCount: 2,
		Quant:      6,
		RegionRects: []ProgressiveRect{{
			Left: 0, Top: 0, Right: 64, Bottom: 64,
		}},
		EncodedData: []byte{1, 2, 3, 4, 5},
	}
}
