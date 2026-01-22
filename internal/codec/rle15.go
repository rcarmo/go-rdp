package codec

// RLE decompression for 15-bit color depth (RGB555)
// This file provides backward-compatible wrappers around the generic implementation.

// ReadPixel15 reads a 15-bit pixel from the buffer
func ReadPixel15(data []byte, idx int) uint16 {
return Pixel15.ReadPixel(data, idx)
}

// WritePixel15 writes a 15-bit pixel to the buffer
func WritePixel15(data []byte, idx int, pixel uint16) {
Pixel15.WritePixel(data, idx, pixel)
}

// WriteFgBgImage15 writes a foreground/background image for 15-bit color
func WriteFgBgImage15(dest []byte, destIdx int, rowDelta int, bitmask byte, fgPel uint16, cBits int, firstLine bool) int {
return writeFgBgImage(Pixel15, dest, destIdx, rowDelta, bitmask, fgPel, cBits, firstLine)
}

// RLEDecompress15 decompresses 15-bit RLE compressed bitmap data
func RLEDecompress15(src []byte, dest []byte, rowDelta int) bool {
return RLEDecompress(Pixel15, src, dest, rowDelta)
}
