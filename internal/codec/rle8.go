package codec

// RLE decompression for 8-bit color depth
// This file provides backward-compatible wrappers around the generic implementation.

// WriteFgBgImage8 writes a foreground/background image for 8-bit color
func WriteFgBgImage8(dest []byte, destIdx int, rowDelta int, bitmask byte, fgPel byte, cBits int, firstLine bool) int {
return writeFgBgImage(Pixel8, dest, destIdx, rowDelta, bitmask, fgPel, cBits, firstLine)
}

// RLEDecompress8 decompresses 8-bit RLE compressed bitmap data
func RLEDecompress8(src []byte, dest []byte, rowDelta int) bool {
return RLEDecompress(Pixel8, src, dest, rowDelta)
}
