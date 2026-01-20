package pdu

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBitmapCacheHostSupportCapabilitySet(t *testing.T) {
	cap := NewBitmapCacheHostSupportCapabilitySet()
	require.NotNil(t, cap)
	require.Equal(t, CapabilitySetTypeBitmapCacheHostSupport, cap.CapabilitySetType)
	require.NotNil(t, cap.BitmapCacheHostSupportCapabilitySet)
}

func TestBitmapCacheHostSupportCapabilitySet_Deserialize(t *testing.T) {
	// cacheVersion (1) + padding1 (1) + padding2 (2)
	input := []byte{0x01, 0x00, 0x00, 0x00}

	cap := &BitmapCacheHostSupportCapabilitySet{}
	err := cap.Deserialize(bytes.NewReader(input))
	require.NoError(t, err)
}

func TestControlCapabilitySet_Serialize(t *testing.T) {
	cap := &ControlCapabilitySet{}
	data := cap.Serialize()

	// Should be 8 bytes: controlFlags, remoteDetachFlag, controlInterest, detachInterest
	require.Len(t, data, 8)
	// controlInterest and detachInterest should be 2
	require.Equal(t, byte(2), data[4])
	require.Equal(t, byte(2), data[6])
}

func TestControlCapabilitySet_Deserialize(t *testing.T) {
	input := []byte{0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x02, 0x00}

	cap := &ControlCapabilitySet{}
	err := cap.Deserialize(bytes.NewReader(input))
	require.NoError(t, err)
}

func TestWindowActivationCapabilitySet_Serialize(t *testing.T) {
	cap := &WindowActivationCapabilitySet{}
	data := cap.Serialize()

	// Should be 8 bytes: helpKeyFlag, helpKeyIndexFlag, helpExtendedKeyFlag, windowManagerKeyFlag
	require.Len(t, data, 8)
}

func TestWindowActivationCapabilitySet_Deserialize(t *testing.T) {
	input := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	cap := &WindowActivationCapabilitySet{}
	err := cap.Deserialize(bytes.NewReader(input))
	require.NoError(t, err)
}

func TestShareCapabilitySet_Serialize(t *testing.T) {
	cap := &ShareCapabilitySet{}
	data := cap.Serialize()

	// Should be 4 bytes: nodeID, pad2octets
	require.Len(t, data, 4)
}

func TestShareCapabilitySet_Deserialize(t *testing.T) {
	input := []byte{0x00, 0x00, 0x00, 0x00}

	cap := &ShareCapabilitySet{}
	err := cap.Deserialize(bytes.NewReader(input))
	require.NoError(t, err)
}

func TestFontCapabilitySet_Serialize(t *testing.T) {
	cap := &FontCapabilitySet{fontSupportFlags: 0x0001}
	data := cap.Serialize()

	// Should be 4 bytes: fontSupportFlags, padding
	require.Len(t, data, 4)
	require.Equal(t, byte(0x01), data[0])
}

func TestFontCapabilitySet_Deserialize(t *testing.T) {
	input := []byte{0x01, 0x00, 0x00, 0x00}

	cap := &FontCapabilitySet{}
	err := cap.Deserialize(bytes.NewReader(input))
	require.NoError(t, err)
	require.Equal(t, uint16(1), cap.fontSupportFlags)
}
