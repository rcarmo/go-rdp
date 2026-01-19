package rdp

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildShareDataHeader_StreamID(t *testing.T) {
	// Verify streamId is always STREAM_LOW (1)
	result := buildShareDataHeader(0x1234, 1000, 0x21, []byte{})
	assert.Equal(t, uint8(1), result[5], "streamId should be STREAM_LOW (1)")
}

func TestBuildShareDataHeader_Padding(t *testing.T) {
	// Verify pad1 is always 0
	result := buildShareDataHeader(0x1234, 1000, 0x21, []byte{})
	assert.Equal(t, uint8(0), result[4], "pad1 should be 0")
}

func TestBuildShareDataHeader_CompressedFields(t *testing.T) {
	// Verify compressedType and compressedLength are 0 (no compression)
	result := buildShareDataHeader(0x1234, 1000, 0x21, []byte{})
	assert.Equal(t, uint8(0), result[9], "compressedType should be 0")
	assert.Equal(t, uint16(0), binary.LittleEndian.Uint16(result[10:12]), "compressedLength should be 0")
}

func TestBuildShareDataHeader_LargeData(t *testing.T) {
	data := make([]byte, 10000)
	result := buildShareDataHeader(0x1234, 1000, 0x21, data)

	// Verify total length is 12 (header) + 10000 (data)
	require.Len(t, result, 12+10000)

	// Verify uncompressedLength is 4 + 10000
	uncompressedLen := binary.LittleEndian.Uint16(result[6:8])
	assert.Equal(t, uint16(4+10000), uncompressedLen)
}

func TestBuildShareControlHeader_Version(t *testing.T) {
	// Verify version is always 1 (encoded in high 12 bits)
	result := buildShareControlHeader(0x0007, 1000, []byte{})

	pduTypeWithVersion := binary.LittleEndian.Uint16(result[2:4])
	version := pduTypeWithVersion >> 4
	assert.Equal(t, uint16(1), version, "version should be 1")
}

func TestBuildShareControlHeader_PDUTypes(t *testing.T) {
	pduTypes := []uint16{
		0x0001, // PDUTYPE_DEMAND_ACTIVE
		0x0003, // PDUTYPE_CONFIRM_ACTIVE
		0x0006, // PDUTYPE_DEACTIVATE_ALL
		0x0007, // PDUTYPE_DATAPDU
	}

	for _, pduType := range pduTypes {
		result := buildShareControlHeader(pduType, 1000, []byte{})
		pduTypeWithVersion := binary.LittleEndian.Uint16(result[2:4])
		extractedType := pduTypeWithVersion & 0x000F
		assert.Equal(t, pduType, extractedType, "PDU type should be preserved")
	}
}

func TestBuildShareControlHeader_AllPduSources(t *testing.T) {
	sources := []uint16{0, 1, 1000, 65535}

	for _, source := range sources {
		result := buildShareControlHeader(0x0007, source, []byte{})
		extractedSource := binary.LittleEndian.Uint16(result[4:6])
		assert.Equal(t, source, extractedSource, "PDU source should be preserved")
	}
}

func TestBuildHeaders_Integration(t *testing.T) {
	// Test building a complete PDU structure
	data := []byte{0x01, 0x02, 0x03, 0x04}
	shareDataHeader := buildShareDataHeader(0x12345678, 1001, 0x21, data)
	shareControlData := buildShareControlHeader(0x0007, 1001, shareDataHeader)

	// Verify structure
	require.GreaterOrEqual(t, len(shareControlData), 6+12+4)

	// Parse total length from share control header
	totalLen := binary.LittleEndian.Uint16(shareControlData[0:2])
	assert.Equal(t, uint16(len(shareControlData)), totalLen)
}

func TestBuildShareDataHeader_VariousPDUTypes(t *testing.T) {
	pduTypes := []uint8{
		0x02, // PDUTYPE2_UPDATE
		0x14, // PDUTYPE2_CONTROL
		0x1F, // PDUTYPE2_SYNCHRONIZE
		0x21, // PDUTYPE2_REFRESH_RECT
		0x28, // PDUTYPE2_FONTLIST
	}

	for _, pduType2 := range pduTypes {
		result := buildShareDataHeader(0x1234, 1000, pduType2, []byte{0x01})
		assert.Equal(t, pduType2, result[8], "pduType2 should be preserved")
	}
}

func TestBuildShareControlHeader_DataIntegrity(t *testing.T) {
	// Verify that data is appended without modification
	originalData := []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE}
	result := buildShareControlHeader(0x0007, 1000, originalData)

	// Data should start at offset 6
	assert.Equal(t, originalData, result[6:])
}

func TestBuildShareDataHeader_DataIntegrity(t *testing.T) {
	// Verify that data is appended without modification
	originalData := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88}
	result := buildShareDataHeader(0x1234, 1000, 0x21, originalData)

	// Data should start at offset 12
	assert.Equal(t, originalData, result[12:])
}

func TestBuildShareDataHeader_BinaryFormat(t *testing.T) {
	result := buildShareDataHeader(0xDEADBEEF, 1000, 0x42, []byte{0xFF})

	// Parse the header manually
	var (
		shareID          uint32
		pad1             uint8
		streamID         uint8
		uncompressedLen  uint16
		pduType2         uint8
		compressedType   uint8
		compressedLength uint16
	)

	buf := bytes.NewReader(result)
	binary.Read(buf, binary.LittleEndian, &shareID)
	binary.Read(buf, binary.LittleEndian, &pad1)
	binary.Read(buf, binary.LittleEndian, &streamID)
	binary.Read(buf, binary.LittleEndian, &uncompressedLen)
	binary.Read(buf, binary.LittleEndian, &pduType2)
	binary.Read(buf, binary.LittleEndian, &compressedType)
	binary.Read(buf, binary.LittleEndian, &compressedLength)

	assert.Equal(t, uint32(0xDEADBEEF), shareID)
	assert.Equal(t, uint8(0), pad1)
	assert.Equal(t, uint8(1), streamID) // STREAM_LOW
	assert.Equal(t, uint16(5), uncompressedLen) // 4 + 1 byte of data
	assert.Equal(t, uint8(0x42), pduType2)
	assert.Equal(t, uint8(0), compressedType)
	assert.Equal(t, uint16(0), compressedLength)
}

func TestBuildShareControlHeader_BinaryFormat(t *testing.T) {
	result := buildShareControlHeader(0x000A, 0x1234, []byte{0xAA})

	// Parse the header manually
	var (
		totalLength       uint16
		pduTypeWithVer    uint16
		pduSource         uint16
	)

	buf := bytes.NewReader(result)
	binary.Read(buf, binary.LittleEndian, &totalLength)
	binary.Read(buf, binary.LittleEndian, &pduTypeWithVer)
	binary.Read(buf, binary.LittleEndian, &pduSource)

	assert.Equal(t, uint16(7), totalLength) // 6 header + 1 data
	assert.Equal(t, uint16(0x001A), pduTypeWithVer) // 0x000A | (1 << 4)
	assert.Equal(t, uint16(0x1234), pduSource)
}
