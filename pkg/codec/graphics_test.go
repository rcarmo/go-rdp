package codec

import "testing"

func TestBitmapCodecGUIDName(t *testing.T) {
	if got := BitmapCodecGUIDName(NSCodecGUID); got != BitmapCodecNameNSCodec {
		t.Fatalf("NSCodec name = %q", got)
	}
	if got := BitmapCodecGUIDName(RemoteFXGUID); got != BitmapCodecNameRemoteFX {
		t.Fatalf("RemoteFX name = %q", got)
	}
	if got := BitmapCodecGUIDName(RemoteFXImageGUID); got != BitmapCodecNameRemoteFXImage {
		t.Fatalf("RemoteFXImage name = %q", got)
	}
	if got := BitmapCodecGUIDName(JPEGCodecGUID); got != BitmapCodecNameJPEG {
		t.Fatalf("JPEG name = %q", got)
	}
	if got := BitmapCodecGUIDName([16]byte{}); got != BitmapCodecNameUnknown {
		t.Fatalf("unknown name = %q", got)
	}
}

func TestEncodeNSCodecRawBGRA(t *testing.T) {
	encoded, ok := EncodeNSCodecRawBGRA([]byte{0x20, 0x40, 0x60, 0xff}, 1, 1, 4)
	if !ok {
		t.Fatal("EncodeNSCodecRawBGRA ok = false")
	}
	if len(encoded) != 23 {
		t.Fatalf("encoded len = %d, want 23", len(encoded))
	}
}

func TestRDPGFXCodecName(t *testing.T) {
	if got := RDPGFXCodecName(RDPGFXCodecClearCodec); got != "ClearCodec" {
		t.Fatalf("ClearCodec name = %q", got)
	}
	if got := RDPGFXCodecName(RDPGFXCodecCAProgressive); got != "CAProgressive" {
		t.Fatalf("CAProgressive name = %q", got)
	}
	if got := RDPGFXCodecName(RDPGFXCodecAVC444); got != "AVC444" {
		t.Fatalf("AVC444 name = %q", got)
	}
	if got := RDPGFXCodecName(RDPGFXCodecAVC444v2); got != "AVC444v2" {
		t.Fatalf("AVC444v2 name = %q", got)
	}
	if got := RDPGFXCodecName(0xffff); got != "Unknown" {
		t.Fatalf("unknown name = %q", got)
	}
}
