package fastpath

// RDPGFX codec identifiers used by WireToSurface PDUs (MS-RDPEGFX).
const (
	RDPGFXCodecUncompressed    uint16 = 0x0000
	RDPGFXCodecCAVideo         uint16 = 0x0003
	RDPGFXCodecClearCodec      uint16 = 0x0008
	RDPGFXCodecCAProgressive   uint16 = 0x0009
	RDPGFXCodecPlanar          uint16 = 0x000A
	RDPGFXCodecAVC420          uint16 = 0x000B
	RDPGFXCodecAlpha           uint16 = 0x000C
	RDPGFXCodecCAProgressiveV2 uint16 = 0x000D
	RDPGFXCodecAVC444          uint16 = 0x000E
	RDPGFXCodecAVC444v2        uint16 = 0x000F
)

// RDPGFXCodecName returns a stable symbolic name for known RDPGFX codec IDs.
func RDPGFXCodecName(codecID uint16) string {
	switch codecID {
	case RDPGFXCodecUncompressed:
		return "Uncompressed"
	case RDPGFXCodecCAVideo:
		return "CAVideo"
	case RDPGFXCodecClearCodec:
		return "ClearCodec"
	case RDPGFXCodecCAProgressive:
		return "CAProgressive"
	case RDPGFXCodecPlanar:
		return "Planar"
	case RDPGFXCodecAVC420:
		return "AVC420"
	case RDPGFXCodecAlpha:
		return "Alpha"
	case RDPGFXCodecCAProgressiveV2:
		return "CAProgressiveV2"
	case RDPGFXCodecAVC444:
		return "AVC444"
	case RDPGFXCodecAVC444v2:
		return "AVC444v2"
	default:
		return "Unknown"
	}
}
