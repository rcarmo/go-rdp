// Package codec implements RLE decompression for RDP bitmap data.
// This implements the Interleaved RLE algorithm as specified in MS-RDPBCGR section 2.2.9.1.1.3.1.2.4.
package codec

// RLE compression order codes
const (
	RegularBgRun         = 0x0
	RegularFgRun         = 0x1
	RegularFgBgImage     = 0x2
	RegularColorRun      = 0x3
	RegularColorImage    = 0x4
	MegaMegaBgRun        = 0xF0
	MegaMegaFgRun        = 0xF1
	MegaMegaFgBgImage    = 0xF2
	MegaMegaColorRun     = 0xF3
	MegaMegaColorImage   = 0xF4
	MegaMegaSetFgRun     = 0xF6
	MegaMegaSetFgBgImage = 0xF7
	MegaMegaDitheredRun  = 0xF8
	LiteSetFgFgRun       = 0xC
	LiteSetFgFgBgImage   = 0xD
	LiteDitheredRun      = 0xE
	SpecialFgBg1         = 0xF9
	SpecialFgBg2         = 0xFA
	White                = 0xFD
	Black                = 0xFE
)

const (
	maskRegularRunLength = 0x1F
	maskLiteRunLength    = 0x0F
	maskSpecialFgBg1     = 0x03
	maskSpecialFgBg2     = 0x05
)

// ExtractCodeID extracts the order code from a header byte
func ExtractCodeID(bOrderHdr byte) uint {
	if (bOrderHdr & 0xC0) != 0xC0 {
		return uint(bOrderHdr >> 5)
	}
	if (bOrderHdr & 0xF0) == 0xF0 {
		return uint(bOrderHdr)
	}
	return uint(bOrderHdr >> 4)
}

// IsRegularCode returns true if the code is a regular order code
func IsRegularCode(code uint) bool {
	switch code {
	case RegularBgRun, RegularFgRun, RegularColorRun, RegularColorImage, RegularFgBgImage:
		return true
	}
	return false
}

// IsLiteCode returns true if the code is a lite order code
func IsLiteCode(code uint) bool {
	switch code {
	case LiteSetFgFgRun, LiteDitheredRun, LiteSetFgFgBgImage:
		return true
	}
	return false
}

// IsMegaMegaCode returns true if the code is a mega-mega order code
func IsMegaMegaCode(code uint) bool {
	switch code {
	case MegaMegaBgRun, MegaMegaFgRun, MegaMegaSetFgRun, MegaMegaDitheredRun,
		MegaMegaColorRun, MegaMegaFgBgImage, MegaMegaSetFgBgImage, MegaMegaColorImage:
		return true
	}
	return false
}

// ExtractRunLength extracts the run length from the source buffer at the given index
func ExtractRunLength(code uint, src []byte, idx int) (length int, nextIdx int) {
	// Bounds check helper
	safeGet := func(i int) byte {
		if i < len(src) {
			return src[i]
		}
		return 0
	}

	if code == RegularFgBgImage {
		length = int(safeGet(idx) & maskRegularRunLength)
		if length == 0 {
			return int(safeGet(idx+1)) + 1, idx + 2
		}
		return length * 8, idx + 1
	}

	if code == LiteSetFgFgBgImage {
		length = int(safeGet(idx) & maskLiteRunLength)
		if length == 0 {
			return int(safeGet(idx+1)) + 1, idx + 2
		}
		return length * 8, idx + 1
	}

	if IsRegularCode(code) {
		length = int(safeGet(idx) & maskRegularRunLength)
		if length == 0 {
			return int(safeGet(idx+1)) + 32, idx + 2
		}
		return length, idx + 1
	}

	if IsLiteCode(code) {
		length = int(safeGet(idx) & maskLiteRunLength)
		if length == 0 {
			return int(safeGet(idx+1)) + 16, idx + 2
		}
		return length, idx + 1
	}

	if IsMegaMegaCode(code) {
		length = int(safeGet(idx+1)) | (int(safeGet(idx+2)) << 8)
		return length, idx + 3
	}

	return 0, idx + 1
}

// FgBgBitmasks is the bitmask table for foreground/background image orders
var FgBgBitmasks = []byte{0x01, 0x02, 0x04, 0x08, 0x10, 0x20, 0x40, 0x80}
