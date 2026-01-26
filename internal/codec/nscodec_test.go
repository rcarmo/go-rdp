package codec

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNSCodecRLEDecompress_RawData(t *testing.T) {
	// When data length equals expected size, return raw data
	data := []byte{0x01, 0x02, 0x03, 0x04}
	result := NSCodecRLEDecompress(data, 4)
	require.Equal(t, data, result)
}

func TestNSCodecRLEDecompress_TooLarge(t *testing.T) {
	// When data length > expected size, return nil
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	result := NSCodecRLEDecompress(data, 4)
	require.Nil(t, result)
}

func TestNSCodecRLEDecompress_TooSmall(t *testing.T) {
	// When data length < 4, return nil
	data := []byte{0x01, 0x02}
	result := NSCodecRLEDecompress(data, 10)
	require.Nil(t, result)
}

func TestNSCodecRLEDecompress_RunSegment(t *testing.T) {
	// Run segment: header byte with 0x80 set
	// 0x83 = run length 3, followed by value byte
	// EndData bytes (4 bytes at end)
	data := []byte{
		0x83,       // Run header: 0x80 | 3
		0xAA,       // Run value
		0x00, 0x00, // EndData
		0x00, 0x00,
	}
	result := NSCodecRLEDecompress(data, 7)
	require.NotNil(t, result)
	// Should have 3x 0xAA followed by EndData
	require.Equal(t, byte(0xAA), result[0])
	require.Equal(t, byte(0xAA), result[1])
	require.Equal(t, byte(0xAA), result[2])
}

func TestNSCodecRLEDecompress_LiteralSegment(t *testing.T) {
	// Literal segment: header byte without 0x80 (compressed run keeps data smaller)
	data := []byte{
		0x84, // Run header: 4 bytes
		0x11, // Run value
		0x02, // Literal length 2
		0x22, 0x33, // Literal bytes
		0xAA, 0xBB, 0xCC, 0xDD, // EndData
	}
	result := NSCodecRLEDecompress(data, 10)
	require.NotNil(t, result)
	require.Equal(t, []byte{0x11, 0x11, 0x11, 0x11, 0x22, 0x33, 0xAA, 0xBB, 0xCC, 0xDD}, result)
}

func TestNSCodecRLEDecompress_TruncatedData(t *testing.T) {
	// Truncated data should return nil
	data := []byte{
		0x03,
		0x11, 0x22, // Missing one literal byte
		0x00, 0x00, 0x00, 0x00,
	}
	result := NSCodecRLEDecompress(data, 10)
	require.Nil(t, result)
}

func TestClampByteNS(t *testing.T) {
	require.Equal(t, byte(0), clampByteNS(-10))
	require.Equal(t, byte(0), clampByteNS(0))
	require.Equal(t, byte(128), clampByteNS(128))
	require.Equal(t, byte(255), clampByteNS(255))
	require.Equal(t, byte(255), clampByteNS(300))
}

func TestAYCoCgToRGBA_NoAlpha(t *testing.T) {
	luma := []byte{128}
	co := []byte{128}
	cg := []byte{128}

	result := AYCoCgToRGBA(luma, co, cg, nil, 1, 1, 1, 1)

	require.Len(t, result, 4)
	require.Equal(t, byte(255), result[3]) // Default alpha = 255
}

func TestDecodeNSCodecToRGBA_TooShort(t *testing.T) {
	// Less than 20 byte header
	data := make([]byte, 10)
	result := DecodeNSCodecToRGBA(data, 10, 10)
	require.Nil(t, result)
}

func TestDecodeNSCodecToRGBA_InvalidColorLoss(t *testing.T) {
	// Valid header size but invalid colorLossLevel (0)
	data := make([]byte, 20)
	data[16] = 0 // colorLossLevel = 0 (invalid, must be 1-7)
	result := DecodeNSCodecToRGBA(data, 10, 10)
	require.Nil(t, result)

	// colorLossLevel = 8 (invalid)
	data[16] = 8
	result = DecodeNSCodecToRGBA(data, 10, 10)
	require.Nil(t, result)
}
