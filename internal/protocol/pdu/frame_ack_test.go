package pdu

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewFrameAcknowledgePDU(t *testing.T) {
	pdu := NewFrameAcknowledgePDU(0x12345678, 1003, 42)

	require.Equal(t, uint32(0x12345678), pdu.ShareDataHeader.ShareID)
	require.Equal(t, uint16(1003), pdu.ShareDataHeader.ShareControlHeader.PDUSource)
	require.Equal(t, uint32(42), pdu.FrameID)
	require.Equal(t, Type2(38), pdu.ShareDataHeader.PDUType2) // PDUTYPE2_FRAME_ACKNOWLEDGE
}

func TestFrameAcknowledgePDU_Serialize(t *testing.T) {
	pdu := NewFrameAcknowledgePDU(0x00010001, 1003, 1)

	data := pdu.Serialize()

	// Should contain ShareControlHeader (6) + ShareDataHeader (12) + FrameID (4) = 22 bytes
	require.Len(t, data, 22)

	// Check frame ID at the end (little-endian)
	require.Equal(t, byte(0x01), data[18])
	require.Equal(t, byte(0x00), data[19])
	require.Equal(t, byte(0x00), data[20])
	require.Equal(t, byte(0x00), data[21])
}
