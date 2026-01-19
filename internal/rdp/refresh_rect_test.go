package rdp

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildShareDataHeader(t *testing.T) {
	tests := []struct {
		name      string
		shareID   uint32
		userID    uint16
		pduType2  uint8
		data      []byte
		checkFunc func(t *testing.T, result []byte)
	}{
		{
			name:     "basic header with empty data",
			shareID:  0x12345678,
			userID:   1001,
			pduType2: 0x21, // PDUTYPE2_REFRESH_RECT
			data:     []byte{},
			checkFunc: func(t *testing.T, result []byte) {
				require.Len(t, result, 12) // 4+1+1+2+1+1+2+0 = 12 bytes

				// Check shareID
				assert.Equal(t, uint32(0x12345678), binary.LittleEndian.Uint32(result[0:4]))
				// Check pad1
				assert.Equal(t, uint8(0), result[4])
				// Check streamId (STREAM_LOW = 1)
				assert.Equal(t, uint8(1), result[5])
				// Check uncompressedLength (4 + 0 = 4)
				assert.Equal(t, uint16(4), binary.LittleEndian.Uint16(result[6:8]))
				// Check pduType2
				assert.Equal(t, uint8(0x21), result[8])
				// Check compressedType
				assert.Equal(t, uint8(0), result[9])
				// Check compressedLength
				assert.Equal(t, uint16(0), binary.LittleEndian.Uint16(result[10:12]))
			},
		},
		{
			name:     "header with data",
			shareID:  0xABCDEF01,
			userID:   2000,
			pduType2: 0x33,
			data:     []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			checkFunc: func(t *testing.T, result []byte) {
				require.Len(t, result, 17) // 12 + 5 bytes of data

				// Check shareID
				assert.Equal(t, uint32(0xABCDEF01), binary.LittleEndian.Uint32(result[0:4]))
				// Check uncompressedLength (4 + 5 = 9)
				assert.Equal(t, uint16(9), binary.LittleEndian.Uint16(result[6:8]))
				// Check pduType2
				assert.Equal(t, uint8(0x33), result[8])
				// Check data
				assert.Equal(t, []byte{0x01, 0x02, 0x03, 0x04, 0x05}, result[12:17])
			},
		},
		{
			name:     "zero shareID",
			shareID:  0,
			userID:   0,
			pduType2: 0x00,
			data:     []byte{0xFF},
			checkFunc: func(t *testing.T, result []byte) {
				require.Len(t, result, 13)
				assert.Equal(t, uint32(0), binary.LittleEndian.Uint32(result[0:4]))
				assert.Equal(t, []byte{0xFF}, result[12:13])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildShareDataHeader(tt.shareID, tt.userID, tt.pduType2, tt.data)
			tt.checkFunc(t, result)
		})
	}
}

func TestBuildShareControlHeader(t *testing.T) {
	tests := []struct {
		name      string
		pduType   uint16
		pduSource uint16
		data      []byte
		checkFunc func(t *testing.T, result []byte)
	}{
		{
			name:      "basic header with empty data",
			pduType:   0x0007, // PDUTYPE_DATAPDU
			pduSource: 1001,
			data:      []byte{},
			checkFunc: func(t *testing.T, result []byte) {
				require.Len(t, result, 6) // 2+2+2+0 = 6 bytes

				// Check totalLength (6 + 0 = 6)
				assert.Equal(t, uint16(6), binary.LittleEndian.Uint16(result[0:2]))
				// Check pduType with version (0x0007 | (1 << 4) = 0x0017)
				assert.Equal(t, uint16(0x0017), binary.LittleEndian.Uint16(result[2:4]))
				// Check pduSource
				assert.Equal(t, uint16(1001), binary.LittleEndian.Uint16(result[4:6]))
			},
		},
		{
			name:      "header with data",
			pduType:   0x0001,
			pduSource: 2000,
			data:      []byte{0xAA, 0xBB, 0xCC, 0xDD},
			checkFunc: func(t *testing.T, result []byte) {
				require.Len(t, result, 10) // 6 + 4 bytes of data

				// Check totalLength (6 + 4 = 10)
				assert.Equal(t, uint16(10), binary.LittleEndian.Uint16(result[0:2]))
				// Check pduType with version (0x0001 | (1 << 4) = 0x0011)
				assert.Equal(t, uint16(0x0011), binary.LittleEndian.Uint16(result[2:4]))
				// Check pduSource
				assert.Equal(t, uint16(2000), binary.LittleEndian.Uint16(result[4:6]))
				// Check data
				assert.Equal(t, []byte{0xAA, 0xBB, 0xCC, 0xDD}, result[6:10])
			},
		},
		{
			name:      "zero values",
			pduType:   0x0000,
			pduSource: 0,
			data:      []byte{},
			checkFunc: func(t *testing.T, result []byte) {
				require.Len(t, result, 6)
				assert.Equal(t, uint16(6), binary.LittleEndian.Uint16(result[0:2]))
				// pduType 0 | (1 << 4) = 0x0010
				assert.Equal(t, uint16(0x0010), binary.LittleEndian.Uint16(result[2:4]))
				assert.Equal(t, uint16(0), binary.LittleEndian.Uint16(result[4:6]))
			},
		},
		{
			name:      "large data",
			pduType:   0x0007,
			pduSource: 65535,
			data:      make([]byte, 100),
			checkFunc: func(t *testing.T, result []byte) {
				require.Len(t, result, 106)
				// Check totalLength (6 + 100 = 106)
				assert.Equal(t, uint16(106), binary.LittleEndian.Uint16(result[0:2]))
				// Check pduSource
				assert.Equal(t, uint16(65535), binary.LittleEndian.Uint16(result[4:6]))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildShareControlHeader(tt.pduType, tt.pduSource, tt.data)
			tt.checkFunc(t, result)
		})
	}
}

func TestBuildShareDataHeader_UserIDIgnored(t *testing.T) {
	// The userID parameter is unused in buildShareDataHeader (it's in the signature but ignored)
	result1 := buildShareDataHeader(0x1234, 1000, 0x21, []byte{0x01})
	result2 := buildShareDataHeader(0x1234, 2000, 0x21, []byte{0x01})

	assert.Equal(t, result1, result2, "userID should be ignored in buildShareDataHeader")
}
