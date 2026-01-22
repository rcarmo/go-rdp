package codec

// RLE decompression for 32-bit color depth (RGBA8888)
// This file provides backward-compatible wrappers around the generic implementation.

// ReadPixel32 reads a 32-bit pixel from the buffer
func ReadPixel32(data []byte, idx int) uint32 {
return Pixel32.ReadPixel(data, idx)
}

// WritePixel32 writes a 32-bit pixel to the buffer
func WritePixel32(data []byte, idx int, pixel uint32) {
Pixel32.WritePixel(data, idx, pixel)
}

// WriteFgBgImage32 writes a foreground/background image for 32-bit color
func WriteFgBgImage32(dest []byte, destIdx int, rowDelta int, bitmask byte, fgPel uint32, cBits int, firstLine bool) int {
return writeFgBgImage(Pixel32, dest, destIdx, rowDelta, bitmask, fgPel, cBits, firstLine)
}

// RLEDecompress32 decompresses 32-bit RLE compressed bitmap data
func RLEDecompress32(src []byte, dest []byte, rowDelta int) bool {
return RLEDecompress(Pixel32, src, dest, rowDelta)
}
