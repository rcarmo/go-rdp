package pdu

import (
	"bytes"
	"encoding/binary"
)

// TS_FRAME_ACKNOWLEDGE_PDU - MS-RDPBCGR 2.2.14.2
// Sent from client to server to acknowledge receipt of a frame
type FrameAcknowledgePDU struct {
	ShareDataHeader ShareDataHeader
	FrameID         uint32
}

// NewFrameAcknowledgePDU creates a new Frame Acknowledge PDU
func NewFrameAcknowledgePDU(shareID uint32, userID uint16, frameID uint32) *FrameAcknowledgePDU {
	return &FrameAcknowledgePDU{
		ShareDataHeader: ShareDataHeader{
			ShareControlHeader: ShareControlHeader{
				PDUType:   TypeData,
				PDUSource: userID,
			},
			ShareID:            shareID,
			StreamID:           0x01, // STREAM_LOW
			UncompressedLength: 4,    // frameID is 4 bytes
			PDUType2:           38,   // PDUTYPE2_FRAME_ACKNOWLEDGE
		},
		FrameID: frameID,
	}
}

// Serialize serializes the Frame Acknowledge PDU to bytes
func (pdu *FrameAcknowledgePDU) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Frame data (just the frameID)
	frameData := make([]byte, 4)
	binary.LittleEndian.PutUint32(frameData, pdu.FrameID)

	// Update lengths
	pdu.ShareDataHeader.UncompressedLength = uint16(len(frameData))                                   // #nosec G115
	totalLen := uint16(6 + 12 + len(frameData)) // #nosec G115 -- ShareControl(6) + ShareData(12) + data
	pdu.ShareDataHeader.ShareControlHeader.TotalLength = totalLen

	// Write header and data
	buf.Write(pdu.ShareDataHeader.Serialize())
	buf.Write(frameData)

	return buf.Bytes()
}
