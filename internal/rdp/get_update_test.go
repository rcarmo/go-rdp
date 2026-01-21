package rdp

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlowPathUpdateTypeConstants(t *testing.T) {
	assert.Equal(t, uint16(0x0000), SlowPathUpdateTypeOrders)
	assert.Equal(t, uint16(0x0001), SlowPathUpdateTypeBitmap)
	assert.Equal(t, uint16(0x0002), SlowPathUpdateTypePalette)
	assert.Equal(t, uint16(0x0003), SlowPathUpdateTypeSynchronize)
}

func TestFastPathUpdateCodeConstants(t *testing.T) {
	assert.Equal(t, uint8(0x01), FastPathUpdateCodeBitmap)
	assert.Equal(t, uint8(0x02), FastPathUpdateCodePalette)
	assert.Equal(t, uint8(0x03), FastPathUpdateCodeSynchronize)
}

func TestUpdateCounter(t *testing.T) {
	// Just verify the variable exists and is accessible
	initialValue := updateCounter
	assert.GreaterOrEqual(t, initialValue, 0)
}

func TestPendingSlowPathUpdate_InitiallyNil(t *testing.T) {
	// The pendingSlowPathUpdate should be nil initially (or after consumed)
	// Create a client and verify its pendingSlowPathUpdate is nil
	client := &Client{}
	assert.Nil(t, client.pendingSlowPathUpdate)
}

func TestClient_handleSlowPathGraphicsUpdate_Bitmap(t *testing.T) {
	// Build bitmap update data
	buf := new(bytes.Buffer)
	
	// Write some bitmap data (number of rectangles followed by rectangle data)
	_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // numberRectangles
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // destLeft
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // destTop
	_ = binary.Write(buf, binary.LittleEndian, uint16(100)) // destRight
	_ = binary.Write(buf, binary.LittleEndian, uint16(100)) // destBottom
	_ = binary.Write(buf, binary.LittleEndian, uint16(100)) // width
	_ = binary.Write(buf, binary.LittleEndian, uint16(100)) // height
	_ = binary.Write(buf, binary.LittleEndian, uint16(16)) // bitsPerPixel
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // flags
	_ = binary.Write(buf, binary.LittleEndian, uint16(4)) // bitmapLength
	buf.Write([]byte{0x01, 0x02, 0x03, 0x04}) // bitmap data

	client := &Client{}
	
	// Prepend updateType for the reader
	inputBuf := new(bytes.Buffer)
	_ = binary.Write(inputBuf, binary.LittleEndian, SlowPathUpdateTypeBitmap)
	inputBuf.Write(buf.Bytes())
	
	result, err := client.handleSlowPathGraphicsUpdate(inputBuf)
	
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.IsType(t, &Update{}, result)
}

func TestClient_handleSlowPathGraphicsUpdate_Palette(t *testing.T) {
	buf := new(bytes.Buffer)
	
	// Write palette data
	_ = binary.Write(buf, binary.LittleEndian, SlowPathUpdateTypePalette)
	_ = binary.Write(buf, binary.LittleEndian, uint16(256)) // numColors
	buf.Write(make([]byte, 256*3)) // RGB values
	
	client := &Client{}
	result, err := client.handleSlowPathGraphicsUpdate(buf)
	
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestClient_handleSlowPathGraphicsUpdate_Synchronize(t *testing.T) {
	buf := new(bytes.Buffer)
	
	_ = binary.Write(buf, binary.LittleEndian, SlowPathUpdateTypeSynchronize)
	// Synchronize has no additional data
	
	client := &Client{}
	result, err := client.handleSlowPathGraphicsUpdate(buf)
	
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestClient_handleSlowPathGraphicsUpdate_Orders(t *testing.T) {
	buf := new(bytes.Buffer)
	
	// Orders (0x0000) is not a known graphics update type for conversion
	_ = binary.Write(buf, binary.LittleEndian, SlowPathUpdateTypeOrders)
	buf.Write([]byte{0x01, 0x02})
	
	client := &Client{}
	result, err := client.handleSlowPathGraphicsUpdate(buf)
	
	require.NoError(t, err)
	// Unknown update types return nil
	assert.Nil(t, result)
}

func TestClient_handleSlowPathGraphicsUpdate_UnknownType(t *testing.T) {
	buf := new(bytes.Buffer)
	
	// Unknown update type
	_ = binary.Write(buf, binary.LittleEndian, uint16(0xFF))
	buf.Write([]byte{0x01, 0x02})
	
	client := &Client{}
	result, err := client.handleSlowPathGraphicsUpdate(buf)
	
	require.NoError(t, err)
	assert.Nil(t, result, "Unknown update types should return nil")
}
