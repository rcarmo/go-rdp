package fastpath

import "testing"

func TestRDPGFXCodecName(t *testing.T) {
	cases := map[uint16]string{
		RDPGFXCodecUncompressed:    "Uncompressed",
		RDPGFXCodecCAVideo:         "CAVideo",
		RDPGFXCodecClearCodec:      "ClearCodec",
		RDPGFXCodecCAProgressive:   "CAProgressive",
		RDPGFXCodecPlanar:          "Planar",
		RDPGFXCodecAVC420:          "AVC420",
		RDPGFXCodecAlpha:           "Alpha",
		RDPGFXCodecCAProgressiveV2: "CAProgressiveV2",
		RDPGFXCodecAVC444:          "AVC444",
		RDPGFXCodecAVC444v2:        "AVC444v2",
		0xffff:                     "Unknown",
	}
	for codecID, want := range cases {
		if got := RDPGFXCodecName(codecID); got != want {
			t.Fatalf("RDPGFXCodecName(0x%04x) = %q, want %q", codecID, got, want)
		}
	}
}
