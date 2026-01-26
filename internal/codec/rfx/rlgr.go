package rfx

// BitStream provides bit-level reading from a byte slice.
// It uses a 32-bit accumulator for efficient bit extraction.
// Bits are read MSB-first (most significant bit first).
type BitStream struct {
	data      []byte
	bytePos   int
	acc       uint32 // 32-bit lookahead accumulator, left-aligned
	bitsInAcc int    // bits available in accumulator
}

// NewBitStream creates a new bit stream reader
func NewBitStream(data []byte) *BitStream {
	bs := &BitStream{
		data:    data,
		bytePos: 0,
	}
	bs.refill()
	return bs
}

// refill loads more bytes into the accumulator (left-aligned)
func (bs *BitStream) refill() {
	for bs.bitsInAcc <= 24 && bs.bytePos < len(bs.data) {
		// Shift accumulator left by 8, add new byte in LSB position
		bs.acc |= uint32(bs.data[bs.bytePos]) << (24 - bs.bitsInAcc)
		bs.bytePos++
		bs.bitsInAcc += 8
	}
}

// ReadBits reads n bits (up to 25) from the stream
func (bs *BitStream) ReadBits(n int) uint32 {
	if n == 0 {
		return 0
	}
	if n > bs.bitsInAcc {
		bs.refill()
	}
	if n > bs.bitsInAcc {
		// Not enough bits remaining, return what we have
		if bs.bitsInAcc == 0 {
			return 0
		}
		result := bs.acc >> (32 - bs.bitsInAcc)
		bs.bitsInAcc = 0
		bs.acc = 0
		return result
	}

	result := bs.acc >> (32 - n)
	bs.acc <<= n
	bs.bitsInAcc -= n
	return result
}

// ReadBit reads a single bit
func (bs *BitStream) ReadBit() uint32 {
	return bs.ReadBits(1)
}

// CountLeadingZeros counts consecutive zero bits (unary prefix) until a 1 is found.
// The terminating 1 bit is consumed.
func (bs *BitStream) CountLeadingZeros() int {
	count := 0
	for {
		if bs.bitsInAcc == 0 {
			bs.refill()
			if bs.bitsInAcc == 0 {
				return count
			}
		}

		// Check top bit
		if (bs.acc & 0x80000000) != 0 {
			// Found a 1 bit, consume it and return
			bs.acc <<= 1
			bs.bitsInAcc--
			return count
		}

		// Consume the 0 bit
		bs.acc <<= 1
		bs.bitsInAcc--
		count++

		// Safety limit to prevent infinite loops on malformed data
		if count > 32000 {
			return count
		}
	}
}

// CountLeadingOnes counts consecutive one bits until a 0 is found.
// The terminating 0 bit is consumed.
func (bs *BitStream) CountLeadingOnes() int {
	count := 0
	for {
		if bs.bitsInAcc == 0 {
			bs.refill()
			if bs.bitsInAcc == 0 {
				return count
			}
		}

		// Check top bit
		if (bs.acc & 0x80000000) == 0 {
			// Found a 0 bit, consume it and return
			bs.acc <<= 1
			bs.bitsInAcc--
			return count
		}

		// Consume the 1 bit
		bs.acc <<= 1
		bs.bitsInAcc--
		count++

		if count > 32000 {
			return count
		}
	}
}

// RemainingBits returns approximate remaining bits in stream
func (bs *BitStream) RemainingBits() int {
	return (len(bs.data)-bs.bytePos)*8 + bs.bitsInAcc
}

// RLGRDecode decodes RLGR-encoded data into coefficient array.
// mode: RLGR1 for Y component, RLGR3 for Cb/Cr
// output: pre-allocated int16 slice of size TilePixels (4096)
func RLGRDecode(data []byte, mode int, output []int16) error {
	if len(output) < TilePixels {
		return ErrBufferTooSmall
	}

	// Clear output buffer
	for i := range output {
		output[i] = 0
	}

	if len(data) == 0 {
		return nil
	}

	bs := NewBitStream(data)

	// Initialize adaptive parameters (MS-RDPRFX 3.1.8.1.7.1)
	k := uint32(1)   // Golomb-Rice parameter for run-length
	kp := uint32(8)  // Scaled k parameter (k = kp >> LSGR)
	kr := uint32(1)  // Golomb-Rice parameter for magnitudes
	krp := uint32(8) // Scaled kr parameter

	idx := 0

	for idx < TilePixels && bs.RemainingBits() > 0 {
		if k != 0 {
			// Run/Literal mode (k > 0)
			// Decode run of zeros followed by a non-zero value

			// Count unary prefix (number of full run-length codes)
			nIdx := bs.CountLeadingZeros()
			if bs.RemainingBits() == 0 {
				return ErrRLGRDecodeError
			}

			// Calculate run length
			runLength := 0
			for i := 0; i < nIdx; i++ {
				runLength += 1 << k

				// Update k parameter (increase after each full run)
				kp += UP_GR
				if kp > KPMAX {
					kp = KPMAX
				}
				k = kp >> LSGR
			}

			// Read remainder bits for partial run
			if k > 0 && bs.RemainingBits() >= int(k) {
				remainder := bs.ReadBits(int(k))
				runLength += int(remainder)
			}

			// Output zeros for run
			for i := 0; i < runLength && idx < TilePixels; i++ {
				output[idx] = 0
				idx++
			}

			if idx >= TilePixels {
				break
			}

			// Decode the non-zero value
			// Sign bit first
			if bs.RemainingBits() == 0 {
				return ErrRLGRDecodeError
			}
			sign := bs.ReadBit()

			// Magnitude using GR coding (count ones, then kr bits)
			nIdx = bs.CountLeadingOnes()
			if bs.RemainingBits() == 0 && nIdx == 0 {
				return ErrRLGRDecodeError
			}

			mag := uint32(0)
			if kr > 0 && bs.RemainingBits() >= int(kr) {
				mag = bs.ReadBits(int(kr))
			}
			mag |= uint32(nIdx) << kr

			// Update kr parameter
			if nIdx == 0 {
				if krp >= 2 {
					krp -= 2
				} else {
					krp = 0
				}
			} else if nIdx > 1 {
				krp += uint32(nIdx)
				if krp > KPMAX {
					krp = KPMAX
				}
			}
			kr = krp >> LSGR

			// Update k parameter (decrease after value)
			if kp >= DN_GR {
				kp -= DN_GR
			} else {
				kp = 0
			}
			k = kp >> LSGR

			// Apply sign and store (magnitude is offset by 1)
			value := int16(mag + 1)
			if sign != 0 {
				value = -value
			}
			output[idx] = value
			idx++

		} else {
			// GR mode (k == 0) - no run-length coding
			if mode == RLGR1 {
				// RLGR1: Single value coding with interleaved sign
			nIdx := bs.CountLeadingOnes()
			if bs.RemainingBits() == 0 && nIdx == 0 {
				return ErrRLGRDecodeError
			}

				mag := uint32(0)
				if kr > 0 && bs.RemainingBits() >= int(kr) {
					mag = bs.ReadBits(int(kr))
				}
				mag |= uint32(nIdx) << kr

				// Update kr
				if nIdx == 0 {
					if krp >= 2 {
						krp -= 2
					} else {
						krp = 0
					}
				} else if nIdx > 1 {
					krp += uint32(nIdx)
					if krp > KPMAX {
						krp = KPMAX
					}
				}
				kr = krp >> LSGR

				// Decode signed value (sign interleaved in LSB)
				var value int16
				if mag == 0 {
					value = 0
					// Update k (increase for zero)
					kp += UQ_GR
					if kp > KPMAX {
						kp = KPMAX
					}
					k = kp >> LSGR
				} else {
					// Sign is LSB of magnitude
					if (mag & 1) != 0 {
						value = -int16((mag + 1) >> 1)
					} else {
						value = int16(mag >> 1)
					}
					// Update k (decrease for non-zero)
					if kp >= DQ_GR {
						kp -= DQ_GR
					} else {
						kp = 0
					}
					k = kp >> LSGR
				}

				output[idx] = value
				idx++

			} else {
				// RLGR3: Paired value coding
				nIdx := bs.CountLeadingOnes()
				if bs.RemainingBits() == 0 && nIdx == 0 {
					return ErrRLGRDecodeError
				}

				code := uint32(0)
				if kr > 0 && bs.RemainingBits() >= int(kr) {
					code = bs.ReadBits(int(kr))
				}
				code |= uint32(nIdx) << kr

				// Update kr
				if nIdx == 0 {
					if krp >= 2 {
						krp -= 2
					} else {
						krp = 0
					}
				} else if nIdx > 1 {
					krp += uint32(nIdx)
					if krp > KPMAX {
						krp = KPMAX
					}
				}
				kr = krp >> LSGR

				// Split code into two values
				// nIdx2 = number of bits needed to represent code
				nIdx2 := 0
				if code > 0 {
					temp := code
					for temp > 0 {
						temp >>= 1
						nIdx2++
					}
				}

				var val1, val2 uint32
				if nIdx2 > 0 {
					if bs.RemainingBits() < nIdx2 {
						return ErrRLGRDecodeError
					}
					val1 = bs.ReadBits(nIdx2)
				}
				val2 = code - val1

				// Update k based on values
				if val1 != 0 && val2 != 0 {
					if kp >= 2*DQ_GR {
						kp -= 2 * DQ_GR
					} else {
						kp = 0
					}
				} else if val1 == 0 && val2 == 0 {
					kp += 2 * UQ_GR
					if kp > KPMAX {
						kp = KPMAX
					}
				}
				k = kp >> LSGR

				// Decode and store first value
				if val1 == 0 {
					output[idx] = 0
				} else if (val1 & 1) != 0 {
					output[idx] = -int16((val1 + 1) >> 1)
				} else {
					output[idx] = int16(val1 >> 1)
				}
				idx++

				if idx >= TilePixels {
					break
				}

				// Decode and store second value
				if val2 == 0 {
					output[idx] = 0
				} else if (val2 & 1) != 0 {
					output[idx] = -int16((val2 + 1) >> 1)
				} else {
					output[idx] = int16(val2 >> 1)
				}
				idx++
			}
		}
	}

	return nil
}
