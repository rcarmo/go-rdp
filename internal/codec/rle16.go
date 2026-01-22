package codec

// RLE decompression for 16-bit color depth (RGB565)
// This file provides backward-compatible wrappers around the generic implementation.

// ReadPixel16 reads a 16-bit pixel from the buffer
func ReadPixel16(data []byte, idx int) uint16 {
return Pixel16.ReadPixel(data, idx)
}

// WritePixel16 writes a 16-bit pixel to the buffer
func WritePixel16(data []byte, idx int, pixel uint16) {
Pixel16.WritePixel(data, idx, pixel)
}

// WriteFgBgImage16 writes a foreground/background image for 16-bit color
func WriteFgBgImage16(dest []byte, destIdx int, rowDelta int, bitmask byte, fgPel uint16, cBits int, firstLine bool) int {
return writeFgBgImage(Pixel16, dest, destIdx, rowDelta, bitmask, fgPel, cBits, firstLine)
}

// RLEDecompress16 decompresses 16-bit RLE compressed bitmap data
func RLEDecompress16(src []byte, dest []byte, rowDelta int) bool {
return RLEDecompress(Pixel16, src, dest, rowDelta)
}
