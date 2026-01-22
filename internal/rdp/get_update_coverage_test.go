package rdp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/rcarmo/rdp-html5/internal/protocol/audio"
	"github.com/rcarmo/rdp-html5/internal/protocol/pdu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetUpdate_WithPendingSlowPath tests returning pending slow-path updates
func TestGetUpdate_WithPendingSlowPath(t *testing.T) {
	pendingUpdate := &Update{Data: []byte{0x01, 0x02, 0x03}}
	client := &Client{
		pendingSlowPathUpdate: pendingUpdate,
	}

	update, err := client.GetUpdate()
	require.NoError(t, err)
	assert.Equal(t, pendingUpdate.Data, update.Data)
	assert.Nil(t, client.pendingSlowPathUpdate)
}

// TestGetX224Update_WithRailChannel tests handling of RAIL channel data
func TestGetX224Update_WithRailChannel(t *testing.T) {
	client := &Client{
		channelIDMap: map[string]uint16{
			"rail":   1006,
			"global": 1003,
		},
		railState: RailStateWaitForData,
	}

	// Create rail channel data
	railData := createTestRailHandshakeData(t)

	mockMCS := &MockMCSLayer{
		ReceiveFunc: func() (uint16, io.Reader, error) {
			return 1006, bytes.NewReader(railData), nil
		},
	}

	client.mcsLayer = mockMCS

	update, err := client.getX224Update()
	require.NoError(t, err)
	assert.Nil(t, update) // RAIL updates return nil
}

// TestGetX224Update_WithAudioChannel tests handling of audio channel data
func TestGetX224Update_WithAudioChannel(t *testing.T) {
	client := &Client{
		channelIDMap: map[string]uint16{
			audio.ChannelRDPSND: 1007,
			"global":            1003,
		},
		audioHandler: NewAudioHandler(nil),
	}

	// Create audio channel data (server audio formats)
	audioData := createTestAudioFormatsData(t)

	mockMCS := &MockMCSLayer{
		ReceiveFunc: func() (uint16, io.Reader, error) {
			return 1007, bytes.NewReader(audioData), nil
		},
	}

	client.mcsLayer = mockMCS

	update, err := client.getX224Update()
	require.NoError(t, err)
	assert.Nil(t, update) // Audio updates return nil
}

// TestGetX224Update_WithDeactivateAll tests handling of deactivate all PDU
func TestGetX224Update_WithDeactivateAll(t *testing.T) {
	client := &Client{
		channelIDMap: map[string]uint16{
			"global": 1003,
		},
	}

	// Create deactivate all PDU
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, uint16(6)) // totalLength
	_ = binary.Write(buf, binary.LittleEndian, uint16(pdu.TypeDeactivateAll))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1001)) // pduSource

	mockMCS := &MockMCSLayer{
		ReceiveFunc: func() (uint16, io.Reader, error) {
			return 1003, bytes.NewReader(buf.Bytes()), nil
		},
	}

	client.mcsLayer = mockMCS

	_, err := client.getX224Update()
	assert.ErrorIs(t, err, pdu.ErrDeactivateAll)
}

// TestGetX224Update_WithBitmapUpdate tests handling of bitmap updates
func TestGetX224Update_WithBitmapUpdate(t *testing.T) {
	client := &Client{
		channelIDMap: map[string]uint16{
			"global": 1003,
		},
	}

	// Create a bitmap update PDU
	buf := createTestBitmapUpdatePDU(t)

	mockMCS := &MockMCSLayer{
		ReceiveFunc: func() (uint16, io.Reader, error) {
			return 1003, bytes.NewReader(buf), nil
		},
	}

	client.mcsLayer = mockMCS

	update, err := client.getX224Update()
	require.NoError(t, err)
	assert.NotNil(t, update)
	assert.NotEmpty(t, update.Data)
}

// TestGetX224Update_WithErrorInfo tests handling of error info PDU
func TestGetX224Update_WithErrorInfo(t *testing.T) {
	client := &Client{
		channelIDMap: map[string]uint16{
			"global": 1003,
		},
	}

	// Create an error info PDU
	buf := createTestErrorInfoPDU(t)

	mockMCS := &MockMCSLayer{
		ReceiveFunc: func() (uint16, io.Reader, error) {
			return 1003, bytes.NewReader(buf), nil
		},
	}

	client.mcsLayer = mockMCS

	update, err := client.getX224Update()
	require.NoError(t, err)
	assert.Nil(t, update) // Error info returns nil update
}

// TestHandleSlowPathGraphicsUpdate_Types tests slow-path graphics update handling
func TestHandleSlowPathGraphicsUpdate_Types(t *testing.T) {
	tests := []struct {
		name       string
		updateType uint16
		expectCode uint8
		expectNil  bool
	}{
		{
			name:       "bitmap update",
			updateType: SlowPathUpdateTypeBitmap,
			expectCode: FastPathUpdateCodeBitmap,
			expectNil:  false,
		},
		{
			name:       "palette update",
			updateType: SlowPathUpdateTypePalette,
			expectCode: FastPathUpdateCodePalette,
			expectNil:  false,
		},
		{
			name:       "synchronize update",
			updateType: SlowPathUpdateTypeSynchronize,
			expectCode: FastPathUpdateCodeSynchronize,
			expectNil:  false,
		},
		{
			name:       "orders update (unsupported)",
			updateType: SlowPathUpdateTypeOrders,
			expectNil:  true,
		},
		{
			name:       "unknown update type",
			updateType: 0xFFFF,
			expectNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{}

			buf := new(bytes.Buffer)
			_ = binary.Write(buf, binary.LittleEndian, tt.updateType)
			// Add some bitmap data
			_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // numberRectangles
			buf.Write([]byte{0x00, 0x00, 0x00, 0x00}) // dummy rect data

			update, err := client.handleSlowPathGraphicsUpdate(buf)
			require.NoError(t, err)

			if tt.expectNil {
				assert.Nil(t, update)
			} else {
				require.NotNil(t, update)
				assert.Equal(t, tt.expectCode, update.Data[0]&0x0f)
			}
		})
	}
}

// TestGetX224Update_ReceiveError tests error handling when receive fails
func TestGetX224Update_ReceiveError(t *testing.T) {
	client := &Client{
		channelIDMap: map[string]uint16{
			"global": 1003,
		},
	}

	mockMCS := &MockMCSLayer{
		ReceiveFunc: func() (uint16, io.Reader, error) {
			return 0, nil, errors.New("receive failed")
		},
	}

	client.mcsLayer = mockMCS

	_, err := client.getX224Update()
	assert.Error(t, err)
}

// Helper functions

func createTestRailHandshakeData(t *testing.T) []byte {
	buf := new(bytes.Buffer)
	
	// Channel PDU header
	_ = binary.Write(buf, binary.LittleEndian, uint32(12)) // length
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x03)) // flags: first|last
	
	// RAIL PDU header
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0005)) // orderType: handshake
	_ = binary.Write(buf, binary.LittleEndian, uint16(4))     // orderLength
	
	// Handshake data
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x0001)) // buildNumber
	
	return buf.Bytes()
}

func createTestAudioFormatsData(t *testing.T) []byte {
	buf := new(bytes.Buffer)
	
	// Channel PDU header
	_ = binary.Write(buf, binary.LittleEndian, uint32(20)) // length
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x03)) // flags: first|last
	
	// RDPSND PDU header
	buf.WriteByte(0x07) // msgType: SNDC_FORMATS
	buf.WriteByte(0x00) // padding
	_ = binary.Write(buf, binary.LittleEndian, uint16(12)) // bodySize
	
	// Server audio formats (minimal)
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))   // dwFlags
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))   // dwVolume
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))   // dwPitch
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))   // wDGramPort
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))   // wNumberOfFormats
	
	return buf.Bytes()
}

func createTestBitmapUpdatePDU(t *testing.T) []byte {
	buf := new(bytes.Buffer)

	// ShareControlHeader
	_ = binary.Write(buf, binary.LittleEndian, uint16(30)) // totalLength
	_ = binary.Write(buf, binary.LittleEndian, uint16(pdu.TypeData))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1001)) // pduSource

	// ShareDataHeader
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x12345678)) // shareId
	_ = binary.Write(buf, binary.LittleEndian, uint8(0))          // padding
	_ = binary.Write(buf, binary.LittleEndian, uint8(1))          // streamId
	_ = binary.Write(buf, binary.LittleEndian, uint16(14))        // uncompressedLength
	_ = binary.Write(buf, binary.LittleEndian, uint8(0x02))       // pduType2 = UPDATE
	_ = binary.Write(buf, binary.LittleEndian, uint8(0))          // compressedType
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))         // compressedLength

	// Update data
	_ = binary.Write(buf, binary.LittleEndian, uint16(SlowPathUpdateTypeBitmap)) // updateType
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))                        // numberRectangles

	return buf.Bytes()
}

func createTestErrorInfoPDU(t *testing.T) []byte {
	buf := new(bytes.Buffer)

	// ShareControlHeader
	_ = binary.Write(buf, binary.LittleEndian, uint16(22)) // totalLength
	_ = binary.Write(buf, binary.LittleEndian, uint16(pdu.TypeData))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1001)) // pduSource

	// ShareDataHeader
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x12345678)) // shareId
	_ = binary.Write(buf, binary.LittleEndian, uint8(0))          // padding
	_ = binary.Write(buf, binary.LittleEndian, uint8(1))          // streamId
	_ = binary.Write(buf, binary.LittleEndian, uint16(14))        // uncompressedLength
	_ = binary.Write(buf, binary.LittleEndian, uint8(0x2F))       // pduType2 = ERROR_INFO
	_ = binary.Write(buf, binary.LittleEndian, uint8(0))          // compressedType
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))         // compressedLength

	// Error info data
	_ = binary.Write(buf, binary.LittleEndian, uint32(0)) // errorInfoType

	return buf.Bytes()
}
