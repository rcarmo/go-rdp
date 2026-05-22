package codec

import "encoding/binary"

// EncodeNSCodecRawBGRA encodes top-down 32-bpp BGRA pixels as an NSCODEC_BITMAP_STREAM
// with raw, non-RLE AYCoCg planes, no chroma subsampling, and colorLossLevel 1.
func EncodeNSCodecRawBGRA(pixels []byte, width, height, stride int) ([]byte, bool) {
	return encodeNSCodecRaw(pixels, width, height, stride, true)
}

// EncodeNSCodecRawRGBA encodes top-down 32-bpp RGBA pixels as an NSCODEC_BITMAP_STREAM
// with raw, non-RLE AYCoCg planes, no chroma subsampling, and colorLossLevel 1.
func EncodeNSCodecRawRGBA(pixels []byte, width, height, stride int) ([]byte, bool) {
	return encodeNSCodecRaw(pixels, width, height, stride, false)
}

func encodeNSCodecRaw(pixels []byte, width, height, stride int, bgra bool) ([]byte, bool) {
	if width <= 0 || height <= 0 {
		return nil, false
	}
	minStride := width * 4
	if stride <= 0 {
		stride = minStride
	}
	if stride < minStride {
		return nil, false
	}
	maxInt := int(^uint(0) >> 1)
	if height > 1 && stride > (maxInt-minStride)/(height-1) {
		return nil, false
	}
	required := minStride
	if height > 1 {
		required += stride * (height - 1)
	}
	if len(pixels) < required || width > maxInt/height {
		return nil, false
	}
	planeSize := width * height
	if planeSize > int(^uint32(0)) || 20 > maxInt-3*planeSize {
		return nil, false
	}
	luma := make([]byte, planeSize)
	orange := make([]byte, planeSize)
	green := make([]byte, planeSize)
	idx := 0
	for y := 0; y < height; y++ {
		row := pixels[y*stride:]
		for x := 0; x < width; x++ {
			si := x * 4
			var r, g, b int
			if bgra {
				b = int(row[si+0])
				g = int(row[si+1])
				r = int(row[si+2])
			} else {
				r = int(row[si+0])
				g = int(row[si+1])
				b = int(row[si+2])
			}
			co := (r - b) / 2
			t := b + co
			cg := g - t
			yv := t + cg/2
			luma[idx] = clampByteNS(yv)
			orange[idx] = clampByteNS(co + 128)
			green[idx] = clampByteNS(cg + 128)
			idx++
		}
	}
	out := make([]byte, 20, 20+3*planeSize)
	binary.LittleEndian.PutUint32(out[0:4], uint32(planeSize))
	binary.LittleEndian.PutUint32(out[4:8], uint32(planeSize))
	binary.LittleEndian.PutUint32(out[8:12], uint32(planeSize))
	binary.LittleEndian.PutUint32(out[12:16], 0)
	out[16] = 1 // colorLossLevel: lossless plane values.
	out[17] = 0 // no chroma subsampling.
	out = append(out, luma...)
	out = append(out, orange...)
	out = append(out, green...)
	return out, true
}
