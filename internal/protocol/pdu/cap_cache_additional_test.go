package pdu

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBitmapCacheCapabilitySetRev2(t *testing.T) {
	set := NewBitmapCacheCapabilitySetRev2()
	require.NotNil(t, set)
	require.Equal(t, CapabilitySetTypeBitmapCacheRev2, set.CapabilitySetType)
	require.NotNil(t, set.BitmapCacheCapabilitySetRev2)
}

func TestBitmapCacheCapabilitySetRev2_DeserializeErrors(t *testing.T) {
	set := &BitmapCacheCapabilitySetRev2{}

	// Empty buffer should error
	err := set.Deserialize(bytes.NewReader(nil))
	require.Error(t, err)

	// Truncated buffer should error
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // CacheFlags only
	err = set.Deserialize(bytes.NewReader(buf.Bytes()))
	require.Error(t, err)
}
