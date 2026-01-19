package auth

import (
	"encoding/binary"
)

// MD4 block size and digest size
const (
	md4BlockSize = 64
	md4Size      = 16
)

func md4(data []byte) []byte {
	var a, b, c, d uint32 = 0x67452301, 0xefcdab89, 0x98badcfe, 0x10325476

	// Padding
	origLen := uint64(len(data))
	data = append(data, 0x80)
	for (len(data)+8)%64 != 0 {
		data = append(data, 0x00)
	}
	// Append original length in bits as 64-bit little-endian
	lenBits := origLen * 8
	lenBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(lenBytes, lenBits)
	data = append(data, lenBytes...)

	// Process each 64-byte block
	for i := 0; i < len(data); i += 64 {
		block := data[i : i+64]
		var x [16]uint32
		for j := 0; j < 16; j++ {
			x[j] = binary.LittleEndian.Uint32(block[j*4:])
		}

		aa, bb, cc, dd := a, b, c, d

		// Round 1
		a = md4Round1(a, b, c, d, x[0], 3)
		d = md4Round1(d, a, b, c, x[1], 7)
		c = md4Round1(c, d, a, b, x[2], 11)
		b = md4Round1(b, c, d, a, x[3], 19)
		a = md4Round1(a, b, c, d, x[4], 3)
		d = md4Round1(d, a, b, c, x[5], 7)
		c = md4Round1(c, d, a, b, x[6], 11)
		b = md4Round1(b, c, d, a, x[7], 19)
		a = md4Round1(a, b, c, d, x[8], 3)
		d = md4Round1(d, a, b, c, x[9], 7)
		c = md4Round1(c, d, a, b, x[10], 11)
		b = md4Round1(b, c, d, a, x[11], 19)
		a = md4Round1(a, b, c, d, x[12], 3)
		d = md4Round1(d, a, b, c, x[13], 7)
		c = md4Round1(c, d, a, b, x[14], 11)
		b = md4Round1(b, c, d, a, x[15], 19)

		// Round 2
		a = md4Round2(a, b, c, d, x[0], 3)
		d = md4Round2(d, a, b, c, x[4], 5)
		c = md4Round2(c, d, a, b, x[8], 9)
		b = md4Round2(b, c, d, a, x[12], 13)
		a = md4Round2(a, b, c, d, x[1], 3)
		d = md4Round2(d, a, b, c, x[5], 5)
		c = md4Round2(c, d, a, b, x[9], 9)
		b = md4Round2(b, c, d, a, x[13], 13)
		a = md4Round2(a, b, c, d, x[2], 3)
		d = md4Round2(d, a, b, c, x[6], 5)
		c = md4Round2(c, d, a, b, x[10], 9)
		b = md4Round2(b, c, d, a, x[14], 13)
		a = md4Round2(a, b, c, d, x[3], 3)
		d = md4Round2(d, a, b, c, x[7], 5)
		c = md4Round2(c, d, a, b, x[11], 9)
		b = md4Round2(b, c, d, a, x[15], 13)

		// Round 3
		a = md4Round3(a, b, c, d, x[0], 3)
		d = md4Round3(d, a, b, c, x[8], 9)
		c = md4Round3(c, d, a, b, x[4], 11)
		b = md4Round3(b, c, d, a, x[12], 15)
		a = md4Round3(a, b, c, d, x[2], 3)
		d = md4Round3(d, a, b, c, x[10], 9)
		c = md4Round3(c, d, a, b, x[6], 11)
		b = md4Round3(b, c, d, a, x[14], 15)
		a = md4Round3(a, b, c, d, x[1], 3)
		d = md4Round3(d, a, b, c, x[9], 9)
		c = md4Round3(c, d, a, b, x[5], 11)
		b = md4Round3(b, c, d, a, x[13], 15)
		a = md4Round3(a, b, c, d, x[3], 3)
		d = md4Round3(d, a, b, c, x[11], 9)
		c = md4Round3(c, d, a, b, x[7], 11)
		b = md4Round3(b, c, d, a, x[15], 15)

		a += aa
		b += bb
		c += cc
		d += dd
	}

	// Output
	result := make([]byte, 16)
	binary.LittleEndian.PutUint32(result[0:], a)
	binary.LittleEndian.PutUint32(result[4:], b)
	binary.LittleEndian.PutUint32(result[8:], c)
	binary.LittleEndian.PutUint32(result[12:], d)
	return result
}

func md4F(x, y, z uint32) uint32 { return (x & y) | (^x & z) }
func md4G(x, y, z uint32) uint32 { return (x & y) | (x & z) | (y & z) }
func md4H(x, y, z uint32) uint32 { return x ^ y ^ z }

func rotl(x uint32, n uint) uint32 { return (x << n) | (x >> (32 - n)) }

func md4Round1(a, b, c, d, x uint32, s uint) uint32 {
	return rotl(a+md4F(b, c, d)+x, s)
}

func md4Round2(a, b, c, d, x uint32, s uint) uint32 {
	return rotl(a+md4G(b, c, d)+x+0x5A827999, s)
}

func md4Round3(a, b, c, d, x uint32, s uint) uint32 {
	return rotl(a+md4H(b, c, d)+x+0x6ED9EBA1, s)
}
