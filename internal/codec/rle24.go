package codec

// RLE decompression for 24-bit color depth (RGB888)
// This file provides backward-compatible wrappers around the generic implementation.

// ReadPixel24 reads a 24-bit pixel from the buffer
func ReadPixel24(data []byte, idx int) uint32 {
return Pixel24.ReadPixel(data, idx)
}

// WritePixel24 writes a 24-bit pixel to the buffer
func WritePixel24(data []byte, idx int, pixel uint32) {
Pixel24.WritePixel(data, idx, pixel)
}

// WriteFgBgImage24 writes a foreground/background image for 24-bit color
func WriteFgBgImage24(dest []byte, destIdx int, rowDelta int, bitmask byte, fgPel uint32, cBits int, firstLine bool) int {
return writeFgBgImage(Pixel24, dest, destIdx, rowDelta, bitmask, fgPel, cBits, firstLine)
}

// RLEDecompress24 decompresses 24-bit RLE compressed bitmap data
func RLEDecompress24(src []byte, dest []byte, rowDelta int) bool {
return RLEDecompress(Pixel24, src, dest, rowDelta)
}
