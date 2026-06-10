package codec

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestParseBitmapCodecCapabilities(t *testing.T) {
	payload := []byte{0x03}
	payload = appendBitmapCodecForTest(payload, NSCodecGUID, 0x11, []byte{1, 1, 3})
	payload = appendBitmapCodecForTest(payload, JPEGCodecGUID, 0x12, nil)
	unknown := [16]byte{0xaa, 0xbb, 0xcc}
	payload = appendBitmapCodecForTest(payload, unknown, 0x13, []byte{0xde, 0xad})

	caps, err := ParseBitmapCodecCapabilities(payload)
	if err != nil {
		t.Fatalf("ParseBitmapCodecCapabilities: %v", err)
	}
	if len(caps) != 3 {
		t.Fatalf("len(caps) = %d, want 3", len(caps))
	}
	if caps[0].Name != BitmapCodecNameNSCodec || caps[0].ID != 0x11 || !bytes.Equal(caps[0].Properties, []byte{1, 1, 3}) {
		t.Fatalf("unexpected NSCodec cap: %#v", caps[0])
	}
	if caps[1].Name != BitmapCodecNameJPEG || caps[1].ID != 0x12 || len(caps[1].Properties) != 0 {
		t.Fatalf("unexpected JPEG cap: %#v", caps[1])
	}
	if caps[2].Name != BitmapCodecNameUnknown || caps[2].ID != 0x13 || !bytes.Equal(caps[2].Properties, []byte{0xde, 0xad}) {
		t.Fatalf("unexpected unknown cap: %#v", caps[2])
	}
	if got, ok := FindBitmapCodec(caps, JPEGCodecGUID); !ok || got.ID != 0x12 {
		t.Fatalf("FindBitmapCodec(JPEG) = %#v,%t", got, ok)
	}
}

func TestParseBitmapCodecCapabilitiesMalformed(t *testing.T) {
	cases := [][]byte{
		nil,
		{0x01},
		appendBitmapCodecForTest([]byte{0x01}, NSCodecGUID, 1, []byte{1})[:20],
		append(appendBitmapCodecForTest([]byte{0x00}, NSCodecGUID, 1, nil), 0xff),
	}
	for _, tc := range cases {
		if _, err := ParseBitmapCodecCapabilities(tc); err == nil {
			t.Fatalf("ParseBitmapCodecCapabilities(%x) error = nil", tc)
		}
	}
}

func TestParseRDPGFXCapsAdvertiseAndAVCFlags(t *testing.T) {
	payload := make([]byte, 2)
	binary.LittleEndian.PutUint16(payload[:2], 2)
	payload = appendRDPGFXCapForTest(payload, RDPGFXCapsVersion81, RDPGFXCapsFlagAVC420Enabled)
	payload = appendRDPGFXCapForTest(payload, RDPGFXCapsVersion106, RDPGFXCapsFlagAVCDisabled)

	caps, err := ParseRDPGFXCapsAdvertise(payload)
	if err != nil {
		t.Fatalf("ParseRDPGFXCapsAdvertise: %v", err)
	}
	if len(caps) != 2 {
		t.Fatalf("len(caps) = %d, want 2", len(caps))
	}
	if caps[0].Version != RDPGFXCapsVersion81 || caps[0].CapsDataLength != 4 || caps[0].Flags != RDPGFXCapsFlagAVC420Enabled {
		t.Fatalf("unexpected first cap: %#v", caps[0])
	}
	if !RDPGFXAVC420Enabled(caps[0].Flags) || RDPGFXAVCDisabled(caps[0].Flags) {
		t.Fatalf("unexpected AVC helpers for flags 0x%x", caps[0].Flags)
	}
	if got := RDPGFXAVCFlagNames(caps[1].Flags); len(got) != 1 || got[0] != "AVC_DISABLED" {
		t.Fatalf("RDPGFXAVCFlagNames = %#v", got)
	}
}

func TestParseRDPGFXCapsAdvertiseMalformed(t *testing.T) {
	tooMany := []byte{0x41, 0x00}
	badLen := []byte{0x01, 0x00, 0x04, 0x00, 0x08, 0x00, 0xff, 0x00, 0x00, 0x00}
	trailing := appendRDPGFXCapForTest([]byte{0x01, 0x00}, RDPGFXCapsVersion8, 0)
	trailing = append(trailing, 0xff)
	for _, tc := range [][]byte{nil, {0x00}, {0x00, 0x00}, tooMany, badLen, trailing} {
		if _, err := ParseRDPGFXCapsAdvertise(tc); err == nil {
			t.Fatalf("ParseRDPGFXCapsAdvertise(%x) error = nil", tc)
		}
	}
}

func TestBuildSetSurfaceBitsDeterministicBytes(t *testing.T) {
	got, err := BuildSetSurfaceBits(Rect{Left: 1, Top: 2, Right: 3, Bottom: 4}, 32, 7, 3, 2, []byte{0xaa, 0xbb})
	if err != nil {
		t.Fatalf("BuildSetSurfaceBits: %v", err)
	}
	want := []byte{
		0x01, 0x00, // cmdType SetSurfaceBits
		0x01, 0x00, 0x02, 0x00, 0x03, 0x00, 0x04, 0x00,
		0x20, 0x00, 0x00, 0x07,
		0x03, 0x00, 0x02, 0x00,
		0x02, 0x00, 0x00, 0x00,
		0xaa, 0xbb,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("BuildSetSurfaceBits = %x, want %x", got, want)
	}
}

func TestBuildRDPGFXFramePDUsDeterministicBytes(t *testing.T) {
	start, err := BuildRDPGFXStartFrame(0x01020304)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{0x0b, 0x00, 0, 0, 0x10, 0, 0, 0, 0, 0, 0, 0, 0x04, 0x03, 0x02, 0x01}; !bytes.Equal(start, want) {
		t.Fatalf("start = %x, want %x", start, want)
	}
	end, err := BuildRDPGFXEndFrame(0x01020304)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{0x0c, 0x00, 0, 0, 0x0c, 0, 0, 0, 0x04, 0x03, 0x02, 0x01}; !bytes.Equal(end, want) {
		t.Fatalf("end = %x, want %x", end, want)
	}
}

func TestBuildRDPGFXSurfacePDUsDeterministicBytes(t *testing.T) {
	create, err := BuildRDPGFXCreateSurface(2, 800, 600, RDPGFXPixelFormatXRGB8888)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{0x09, 0, 0, 0, 0x0f, 0, 0, 0, 0x02, 0, 0x20, 0x03, 0x58, 0x02, 0x20}; !bytes.Equal(create, want) {
		t.Fatalf("create = %x, want %x", create, want)
	}
	mapped, err := BuildRDPGFXMapSurfaceToOutput(2, 10, 20)
	if err != nil {
		t.Fatal(err)
	}
	if want := []byte{0x0f, 0, 0, 0, 0x14, 0, 0, 0, 0x02, 0, 0, 0, 0x0a, 0, 0, 0, 0x14, 0, 0, 0}; !bytes.Equal(mapped, want) {
		t.Fatalf("map = %x, want %x", mapped, want)
	}
}

func TestBuildRDPGFXWireToSurface1DeterministicBytes(t *testing.T) {
	got, err := BuildRDPGFXWireToSurface1(1, RDPGFXCodecPlanar, RDPGFXPixelFormatXRGB8888, Rect{Right: 2, Bottom: 1}, []byte{0xab, 0xcd})
	if err != nil {
		t.Fatalf("BuildRDPGFXWireToSurface1: %v", err)
	}
	want := []byte{
		0x01, 0x00, 0x00, 0x00, 0x1b, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x0a, 0x00, 0x20,
		0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01, 0x00,
		0x02, 0x00, 0x00, 0x00,
		0xab, 0xcd,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("wire = %x, want %x", got, want)
	}
}

func TestBuildersRejectInvalidBounds(t *testing.T) {
	if _, err := BuildSetSurfaceBits(Rect{}, 32, 0, 1, 1, []byte{1}); err == nil {
		t.Fatal("BuildSetSurfaceBits accepted zero codec")
	}
	if _, err := BuildSetSurfaceBits(Rect{}, 32, 1, 0, 1, []byte{1}); err == nil {
		t.Fatal("BuildSetSurfaceBits accepted zero width")
	}
	if _, err := BuildSetSurfaceBits(Rect{}, 32, 1, 1, 1, nil); err == nil {
		t.Fatal("BuildSetSurfaceBits accepted empty payload")
	}
	if _, err := BuildRDPGFXCreateSurface(0, 0, 1, RDPGFXPixelFormatXRGB8888); err == nil {
		t.Fatal("BuildRDPGFXCreateSurface accepted zero width")
	}
	if _, err := BuildRDPGFXWireToSurface1(0, RDPGFXCodecPlanar, RDPGFXPixelFormatXRGB8888, Rect{}, nil); err == nil {
		t.Fatal("BuildRDPGFXWireToSurface1 accepted empty payload")
	}
}

func appendBitmapCodecForTest(dst []byte, guid [16]byte, id uint8, props []byte) []byte {
	dst = append(dst, guid[:]...)
	dst = append(dst, id, byte(len(props)), byte(len(props)>>8))
	return append(dst, props...)
}

func appendRDPGFXCapForTest(dst []byte, version, flags uint32) []byte {
	var buf [12]byte
	binary.LittleEndian.PutUint32(buf[0:4], version)
	binary.LittleEndian.PutUint32(buf[4:8], 4)
	binary.LittleEndian.PutUint32(buf[8:12], flags)
	return append(dst, buf[:]...)
}
