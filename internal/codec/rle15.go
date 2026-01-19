package codec

// RLE decompression for 15-bit color depth
// 15-bit uses same RLE format as 16-bit (2 bytes per pixel)
// The pixel format is RGB555 instead of RGB565

// RLEDecompress15 decompresses 15-bit RLE compressed bitmap data
// Uses same algorithm as 16-bit since pixel size is 2 bytes
func RLEDecompress15(src []byte, dest []byte, rowDelta int) bool {
	// 15-bit uses exact same RLE format as 16-bit
	return RLEDecompress16(src, dest, rowDelta)
}
