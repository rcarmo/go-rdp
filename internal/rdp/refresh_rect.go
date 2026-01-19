package rdp

import (
	"bytes"
	"encoding/binary"
)

// sendRefreshRect sends a Refresh Rect PDU to request a full screen update
// [MS-RDPBCGR] 2.2.11.2 Client Refresh Rect PDU
func (c *Client) sendRefreshRect() error {
	// Build the Refresh Rect PDU data
	// numberOfAreas (1 byte) + pad3Octets (3 bytes) + areasToRefresh (variable)
	refreshData := new(bytes.Buffer)

	// numberOfAreas - requesting 1 rectangle (full screen)
	binary.Write(refreshData, binary.LittleEndian, uint8(1))
	// pad3Octets
	binary.Write(refreshData, binary.LittleEndian, uint8(0))
	binary.Write(refreshData, binary.LittleEndian, uint8(0))
	binary.Write(refreshData, binary.LittleEndian, uint8(0))

	// Inclusive Rectangle - full screen
	// left (2 bytes)
	binary.Write(refreshData, binary.LittleEndian, uint16(0))
	// top (2 bytes)
	binary.Write(refreshData, binary.LittleEndian, uint16(0))
	// right (2 bytes) - inclusive, so width-1
	binary.Write(refreshData, binary.LittleEndian, uint16(c.desktopWidth-1))
	// bottom (2 bytes) - inclusive, so height-1
	binary.Write(refreshData, binary.LittleEndian, uint16(c.desktopHeight-1))

	// Build the Share Data Header
	// PDUTYPE_DATAPDU (0x0007)
	// pduType2 = 0x21 (PDUTYPE2_REFRESH_RECT)
	shareDataHeaderData := buildShareDataHeader(c.shareID, c.userID, 0x21, refreshData.Bytes())

	// Build Share Control Header
	// PDUTYPE_DATAPDU = 0x0007
	shareControlData := buildShareControlHeader(0x0007, c.userID, shareDataHeaderData)

	return c.mcsLayer.Send(c.userID, c.channelIDMap["global"], shareControlData)
}

// buildShareDataHeader creates a Share Data Header for a PDU
func buildShareDataHeader(shareID uint32, _ uint16, pduType2 uint8, data []byte) []byte {
	buf := new(bytes.Buffer)

	// shareID (4 bytes)
	binary.Write(buf, binary.LittleEndian, shareID)
	// pad1 (1 byte)
	binary.Write(buf, binary.LittleEndian, uint8(0))
	// streamId (1 byte) - STREAM_LOW = 1
	binary.Write(buf, binary.LittleEndian, uint8(1))
	// uncompressedLength (2 bytes) - includes pduType2, compressedType, compressedLength
	uncompressedLen := uint16(4 + len(data)) // pduType2 + compressedType + compressedLength + data
	binary.Write(buf, binary.LittleEndian, uncompressedLen)
	// pduType2 (1 byte)
	binary.Write(buf, binary.LittleEndian, pduType2)
	// compressedType (1 byte)
	binary.Write(buf, binary.LittleEndian, uint8(0))
	// compressedLength (2 bytes)
	binary.Write(buf, binary.LittleEndian, uint16(0))
	// data
	buf.Write(data)

	return buf.Bytes()
}

// buildShareControlHeader creates a Share Control Header wrapping a Share Data Header
func buildShareControlHeader(pduType uint16, pduSource uint16, data []byte) []byte {
	buf := new(bytes.Buffer)

	// totalLength (2 bytes) - includes this header
	totalLen := uint16(6 + len(data)) // 2 + 2 + 2 + data
	binary.Write(buf, binary.LittleEndian, totalLen)
	// pduType (2 bytes) - low 4 bits = pduType, high 12 bits = version (1)
	pduTypeWithVersion := pduType | (1 << 4) // version 1
	binary.Write(buf, binary.LittleEndian, pduTypeWithVersion)
	// pduSource (2 bytes)
	binary.Write(buf, binary.LittleEndian, pduSource)
	// data
	buf.Write(data)

	return buf.Bytes()
}
