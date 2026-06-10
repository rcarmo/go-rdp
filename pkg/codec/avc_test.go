package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestNormalizeH264AnnexBAndNALTypes(t *testing.T) {
	annexB := []byte{0, 0, 0, 1, 0x67, 1, 2, 0, 0, 1, 0x68, 3, 0, 0, 1, 0x65, 4}
	got, err := NormalizeH264AnnexB(annexB)
	if err != nil || !bytes.Equal(got, annexB) {
		t.Fatalf("NormalizeH264AnnexB annexB got=%x err=%v", got, err)
	}
	if !H264AnnexBContainsNALType(got, 7) || !H264AnnexBContainsNALType(got, 8) || !H264AnnexBContainsNALType(got, 5) || H264AnnexBContainsNALType(got, 1) {
		t.Fatalf("NAL type detection failed for %x", got)
	}
	lengthPrefixed := []byte{0, 0, 0, 2, 0x67, 1, 0, 0, 0, 2, 0x65, 2}
	got, err = NormalizeH264AnnexB(lengthPrefixed)
	if err != nil {
		t.Fatalf("NormalizeH264AnnexB length-prefixed: %v", err)
	}
	want := []byte{0, 0, 0, 1, 0x67, 1, 0, 0, 0, 1, 0x65, 2}
	if !bytes.Equal(got, want) {
		t.Fatalf("length-prefixed normalized=%x want=%x", got, want)
	}
	if _, err := NormalizeH264AnnexB([]byte{0, 0, 0, 9, 1}); err == nil {
		t.Fatal("accepted malformed length-prefixed AU")
	}
}

func TestBuildAVC420BitmapStream(t *testing.T) {
	au := []byte{0, 0, 0, 1, 0x65, 0xaa}
	payload, err := BuildAVC420BitmapStream(au, 16, 8)
	if err != nil {
		t.Fatalf("BuildAVC420BitmapStream: %v", err)
	}
	wantPrefix := []byte{1, 0, 0, 0, 0, 0, 0, 0, 16, 0, 8, 0, 0, 0}
	if !bytes.Equal(payload[:len(wantPrefix)], wantPrefix) || !bytes.Equal(payload[len(wantPrefix):], au) {
		t.Fatalf("payload=%x", payload)
	}
	again, err := BuildAVC420BitmapStream(au, 16, 8)
	if err != nil || !bytes.Equal(payload, again) {
		t.Fatalf("AVC420 deterministic err=%v", err)
	}
	if _, err := BuildAVC420BitmapStream(nil, 16, 8); err == nil {
		t.Fatal("accepted empty AVC420 AU")
	}
}

func TestBuildAVC420WireToSurface(t *testing.T) {
	wire, err := BuildAVC420WireToSurface(1, RDPGFXPixelFormatXRGB8888, Rect{Right: 16, Bottom: 8}, []byte{0, 0, 1, 0x65}, 16, 8)
	if err != nil {
		t.Fatalf("BuildAVC420WireToSurface: %v", err)
	}
	if binary.LittleEndian.Uint16(wire[10:12]) != RDPGFXCodecAVC420 {
		t.Fatalf("codec=0x%04x", binary.LittleEndian.Uint16(wire[10:12]))
	}
}

func TestValidateAVC444Input(t *testing.T) {
	valid := avc444Seed(false)
	if err := ValidateAVC444Input(valid); err != nil {
		t.Fatalf("ValidateAVC444Input valid: %v", err)
	}
	cases := []AVC444Input{
		{},
		{Width: 16, Height: 16, BaseLayer: H264AccessUnit{KeyFrame: false, Data: []byte{1}}, AuxLayer: valid.AuxLayer, RegionRects: valid.RegionRects},
		{Width: 16, Height: 16, BaseLayer: valid.BaseLayer, AuxLayer: H264AccessUnit{}, RegionRects: valid.RegionRects},
		{Width: 16, Height: 16, BaseLayer: valid.BaseLayer, AuxLayer: valid.AuxLayer},
		{Width: 16, Height: 16, BaseLayer: valid.BaseLayer, AuxLayer: valid.AuxLayer, RegionRects: []ProgressiveRect{{Right: 17, Bottom: 16}}},
	}
	for _, tc := range cases {
		if err := ValidateAVC444Input(tc); err == nil {
			t.Fatalf("ValidateAVC444Input(%#v) err=nil", tc)
		}
	}
}

func TestBuildAVC444BitmapStreamV1V2(t *testing.T) {
	v1, err := BuildAVC444BitmapStream(avc444Seed(false))
	if err != nil {
		t.Fatalf("BuildAVC444BitmapStream v1: %v", err)
	}
	v2, err := BuildAVC444BitmapStream(avc444Seed(true))
	if err != nil {
		t.Fatalf("BuildAVC444BitmapStream v2: %v", err)
	}
	if bytes.Equal(v1, v2) {
		t.Fatal("v1/v2 payloads should differ")
	}
	if binary.LittleEndian.Uint32(v1[0:4]) != 1 || binary.LittleEndian.Uint16(v1[4+8+4+4:4+8+4+4+2]) != 0 {
		t.Fatalf("unexpected v1 header=%x", v1[:22])
	}
	if binary.LittleEndian.Uint16(v2[4+8+4+4:4+8+4+4+2]) != 1 {
		t.Fatalf("unexpected v2 flags=%x", v2[:22])
	}
	again, err := BuildAVC444BitmapStream(avc444Seed(false))
	if err != nil || !bytes.Equal(v1, again) {
		t.Fatalf("AVC444 deterministic err=%v", err)
	}
}

func TestBuildAVC444WireToSurface(t *testing.T) {
	wire, err := BuildAVC444WireToSurface(2, RDPGFXPixelFormatXRGB8888, Rect{Right: 16, Bottom: 16}, avc444Seed(false))
	if err != nil {
		t.Fatalf("BuildAVC444WireToSurface: %v", err)
	}
	if binary.LittleEndian.Uint16(wire[10:12]) != RDPGFXCodecAVC444 {
		t.Fatalf("codec=0x%04x", binary.LittleEndian.Uint16(wire[10:12]))
	}
	wire, err = BuildAVC444WireToSurface(2, RDPGFXPixelFormatXRGB8888, Rect{Right: 16, Bottom: 16}, avc444Seed(true))
	if err != nil {
		t.Fatalf("BuildAVC444WireToSurface v2: %v", err)
	}
	if binary.LittleEndian.Uint16(wire[10:12]) != RDPGFXCodecAVC444v2 {
		t.Fatalf("v2 codec=0x%04x", binary.LittleEndian.Uint16(wire[10:12]))
	}
}

func TestValidateH264AccessUnitBatch(t *testing.T) {
	units := []H264AccessUnit{{Data: []byte{1}}, {Data: []byte{2}}}
	if err := ValidateH264AccessUnitBatch(units); err != nil {
		t.Fatalf("ValidateH264AccessUnitBatch: %v", err)
	}
	if err := ValidateH264AccessUnitBatch(nil); err == nil {
		t.Fatal("accepted empty batch")
	}
	if err := ValidateH264AccessUnitBatch([]H264AccessUnit{{}, {}, {}, {}, {}}); err == nil {
		t.Fatal("accepted too many units")
	}
}

func avc444Seed(v2 bool) AVC444Input {
	return AVC444Input{
		Width:     16,
		Height:    16,
		BaseLayer: H264AccessUnit{KeyFrame: true, Data: []byte{0, 0, 0, 1, 0x65, 0xaa}},
		AuxLayer:  H264AccessUnit{Data: []byte{0, 0, 0, 1, 0x61, 0xbb}},
		RegionRects: []ProgressiveRect{{
			Left: 0, Top: 0, Right: 16, Bottom: 16,
		}},
		UseV2: v2,
	}
}
