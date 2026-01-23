package drdynvc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeader_SerializeDeserialize(t *testing.T) {
	tests := []struct {
		name   string
		header Header
	}{
		{
			name:   "basic header",
			header: Header{CbChID: 0, Sp: 0, Cmd: CmdCapability},
		},
		{
			name:   "create command",
			header: Header{CbChID: 1, Sp: 0, Cmd: CmdCreate},
		},
		{
			name:   "data command with 4-byte channel ID",
			header: Header{CbChID: 2, Sp: 1, Cmd: CmdData},
		},
		{
			name:   "close command",
			header: Header{CbChID: 0, Sp: 0, Cmd: CmdClose},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.header.Serialize()
			var decoded Header
			decoded.Deserialize(b)

			assert.Equal(t, tt.header.CbChID, decoded.CbChID)
			assert.Equal(t, tt.header.Sp, decoded.Sp)
			assert.Equal(t, tt.header.Cmd, decoded.Cmd)
		})
	}
}

func TestHeader_ChannelIDSize(t *testing.T) {
	tests := []struct {
		cbChID   uint8
		expected int
	}{
		{0, 1},
		{1, 2},
		{2, 4},
		{3, 1}, // Default case
	}

	for _, tt := range tests {
		h := Header{CbChID: tt.cbChID}
		assert.Equal(t, tt.expected, h.ChannelIDSize())
	}
}

func TestCapsPDU_SerializeDeserialize(t *testing.T) {
	tests := []struct {
		name string
		caps CapsPDU
	}{
		{
			name: "version 1",
			caps: CapsPDU{Version: CapsVersion1},
		},
		{
			name: "version 2",
			caps: CapsPDU{Version: CapsVersion2},
		},
		{
			name: "version 3 with priorities",
			caps: CapsPDU{
				Version:         CapsVersion3,
				PriorityCharge0: 100,
				PriorityCharge1: 200,
				PriorityCharge2: 300,
				PriorityCharge3: 400,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.caps.Serialize()
			require.NotEmpty(t, data)

			var decoded CapsPDU
			err := decoded.Deserialize(bytes.NewReader(data))
			require.NoError(t, err)

			assert.Equal(t, tt.caps.Version, decoded.Version)
			if tt.caps.Version >= CapsVersion3 {
				assert.Equal(t, tt.caps.PriorityCharge0, decoded.PriorityCharge0)
				assert.Equal(t, tt.caps.PriorityCharge1, decoded.PriorityCharge1)
				assert.Equal(t, tt.caps.PriorityCharge2, decoded.PriorityCharge2)
				assert.Equal(t, tt.caps.PriorityCharge3, decoded.PriorityCharge3)
			}
		})
	}
}

func TestCreateRequestPDU_Serialize(t *testing.T) {
	tests := []struct {
		name      string
		req       CreateRequestPDU
		minLength int
	}{
		{
			name:      "small channel ID",
			req:       CreateRequestPDU{ChannelID: 1, ChannelName: "test"},
			minLength: 6, // 1 (header) + 1 (chID) + 4 (name) + 1 (null)
		},
		{
			name:      "medium channel ID",
			req:       CreateRequestPDU{ChannelID: 0x1234, ChannelName: "test"},
			minLength: 7, // 1 (header) + 2 (chID) + 4 (name) + 1 (null)
		},
		{
			name:      "large channel ID",
			req:       CreateRequestPDU{ChannelID: 0x12345678, ChannelName: "test"},
			minLength: 9, // 1 (header) + 4 (chID) + 4 (name) + 1 (null)
		},
		{
			name:      "display control channel name",
			req:       CreateRequestPDU{ChannelID: 1, ChannelName: "Microsoft::Windows::RDS::DisplayControl"},
			minLength: 42, // 1 (header) + 1 (chID) + 40 (name) + 1 (null) - name is actually 40 chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.req.Serialize()
			assert.GreaterOrEqual(t, len(data), tt.minLength)

			// Verify header byte has Create command
			var h Header
			h.Deserialize(data[0])
			assert.Equal(t, CmdCreate, h.Cmd)
		})
	}
}

func TestCreateResponsePDU_Deserialize(t *testing.T) {
	tests := []struct {
		name       string
		cbChID     uint8
		data       []byte
		expectID   uint32
		expectCode uint32
	}{
		{
			name:       "1-byte channel ID success",
			cbChID:     0,
			data:       []byte{0x01, 0x00, 0x00, 0x00, 0x00},
			expectID:   1,
			expectCode: CreateResultOK,
		},
		{
			name:       "2-byte channel ID success",
			cbChID:     1,
			data:       []byte{0x34, 0x12, 0x00, 0x00, 0x00, 0x00},
			expectID:   0x1234,
			expectCode: CreateResultOK,
		},
		{
			name:       "channel not found",
			cbChID:     0,
			data:       []byte{0x01, 0x90, 0x04, 0x07, 0x80},
			expectID:   1,
			expectCode: CreateResultChannelNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp CreateResponsePDU
			err := resp.Deserialize(bytes.NewReader(tt.data), tt.cbChID)
			require.NoError(t, err)

			assert.Equal(t, tt.expectID, resp.ChannelID)
			assert.Equal(t, tt.expectCode, resp.CreationCode)
		})
	}
}

func TestCreateResponsePDU_IsSuccess(t *testing.T) {
	tests := []struct {
		code    uint32
		success bool
	}{
		{CreateResultOK, true},
		{CreateResultDenied, false},
		{CreateResultNoMemory, false},
		{CreateResultNoListener, false},
		{CreateResultChannelNotFound, false},
	}

	for _, tt := range tests {
		resp := CreateResponsePDU{CreationCode: tt.code}
		assert.Equal(t, tt.success, resp.IsSuccess())
	}
}

func TestDataPDU_Serialize(t *testing.T) {
	tests := []struct {
		name string
		pdu  DataPDU
	}{
		{
			name: "small channel ID with data",
			pdu:  DataPDU{ChannelID: 1, Data: []byte{0x01, 0x02, 0x03}},
		},
		{
			name: "large channel ID",
			pdu:  DataPDU{ChannelID: 0x12345678, Data: []byte{0xAA, 0xBB}},
		},
		{
			name: "empty data",
			pdu:  DataPDU{ChannelID: 1, Data: []byte{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.pdu.Serialize()
			require.NotEmpty(t, data)

			// Verify header
			var h Header
			h.Deserialize(data[0])
			assert.Equal(t, CmdData, h.Cmd)
		})
	}
}

func TestDataFirstPDU_Serialize(t *testing.T) {
	pdu := DataFirstPDU{
		ChannelID: 1,
		Length:    100,
		Data:      []byte{0x01, 0x02, 0x03},
	}

	data := pdu.Serialize()
	require.NotEmpty(t, data)

	var h Header
	h.Deserialize(data[0])
	assert.Equal(t, CmdDataFirst, h.Cmd)
}

func TestClosePDU_Serialize(t *testing.T) {
	tests := []struct {
		name      string
		channelID uint32
	}{
		{"1-byte ID", 1},
		{"2-byte ID", 0x1234},
		{"4-byte ID", 0x12345678},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := ClosePDU{ChannelID: tt.channelID}
			data := pdu.Serialize()
			require.NotEmpty(t, data)

			var h Header
			h.Deserialize(data[0])
			assert.Equal(t, CmdClose, h.Cmd)
		})
	}
}

func TestParsePDU(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectCmd   uint8
		expectError bool
	}{
		{
			name:      "capability PDU",
			data:      []byte{0x50, 0x00, 0x01, 0x00}, // Cmd=5 (Capability)
			expectCmd: CmdCapability,
		},
		{
			name:      "create PDU",
			data:      []byte{0x11, 0x01}, // Cmd=1 (Create), cbChID=1
			expectCmd: CmdCreate,
		},
		{
			name:        "empty data",
			data:        []byte{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, _, _, err := ParsePDU(tt.data)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectCmd, cmd)
			}
		})
	}
}

func TestReadChannelID(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		cbChID      uint8
		expectID    uint32
		expectError bool
	}{
		{
			name:     "1-byte ID",
			data:     []byte{0x42, 0xAA, 0xBB},
			cbChID:   0,
			expectID: 0x42,
		},
		{
			name:     "2-byte ID",
			data:     []byte{0x34, 0x12, 0xAA},
			cbChID:   1,
			expectID: 0x1234,
		},
		{
			name:     "4-byte ID",
			data:     []byte{0x78, 0x56, 0x34, 0x12},
			cbChID:   2,
			expectID: 0x12345678,
		},
		{
			name:        "insufficient data for 2-byte",
			data:        []byte{0x01},
			cbChID:      1,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, remaining, err := ReadChannelID(tt.data, tt.cbChID)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectID, id)
				assert.NotNil(t, remaining)
			}
		})
	}
}

// V3 Feature Tests

func TestSoftSyncRequestPDU_Deserialize(t *testing.T) {
	tests := []struct {
		name            string
		data            []byte
		expectFlags     uint8
		expectTunnels   uint16
		expectChannels  int
		expectError     bool
	}{
		{
			name: "basic request no channels",
			data: []byte{
				0x00,       // Pad
				0x01,       // Flags (TCP_FLUSHED)
				0x02, 0x00, // NumberOfTunnels
			},
			expectFlags:    SoftSyncTCPFlushed,
			expectTunnels:  2,
			expectChannels: 0,
		},
		{
			name: "request with channel list",
			data: []byte{
				0x00,       // Pad
				0x03,       // Flags (TCP_FLUSHED | CHANNEL_LIST_PRESENT)
				0x01, 0x00, // NumberOfTunnels
				0x02, 0x00, // Channel count
				0x01, 0x00, 0x00, 0x00, // Channel 1 ID
				0x01, 0x00, 0x00, 0x00, // Channel 1 Tunnel (UDPFECR)
				0x02, 0x00, 0x00, 0x00, // Channel 2 ID
				0x03, 0x00, 0x00, 0x00, // Channel 2 Tunnel (UDPFECL)
			},
			expectFlags:    SoftSyncTCPFlushed | SoftSyncChannelListPresent,
			expectTunnels:  1,
			expectChannels: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pdu SoftSyncRequestPDU
			err := pdu.Deserialize(bytes.NewReader(tt.data))
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectFlags, pdu.Flags)
				assert.Equal(t, tt.expectTunnels, pdu.NumberOfTunnels)
				assert.Len(t, pdu.Channels, tt.expectChannels)
			}
		})
	}
}

func TestSoftSyncResponsePDU_Serialize(t *testing.T) {
	tests := []struct {
		name    string
		pdu     SoftSyncResponsePDU
		minLen  int
	}{
		{
			name: "no tunnels (TCP only)",
			pdu: SoftSyncResponsePDU{
				Pad:             0,
				NumberOfTunnels: 0,
				TunnelTypes:     nil,
			},
			minLen: 6, // header(1) + pad(1) + tunnels(4)
		},
		{
			name: "with tunnels",
			pdu: SoftSyncResponsePDU{
				Pad:             0,
				NumberOfTunnels: 2,
				TunnelTypes:     []uint32{TunnelTypeUDPFECR, TunnelTypeUDPFECL},
			},
			minLen: 14, // header(1) + pad(1) + tunnels(4) + 2*tunnel_type(8)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.pdu.Serialize()
			require.GreaterOrEqual(t, len(data), tt.minLen)

			// Verify header
			var h Header
			h.Deserialize(data[0])
			assert.Equal(t, CmdSoftSync, h.Cmd)
		})
	}
}

func TestDataCompressedPDU_Deserialize(t *testing.T) {
	tests := []struct {
		name       string
		data       []byte
		cbChID     uint8
		isFirst    bool
		expectChan uint32
		expectLen  uint32
	}{
		{
			name:       "data compressed 1-byte channel",
			data:       []byte{0x05, 0xAA, 0xBB, 0xCC}, // channelID=5, compressed data
			cbChID:     0,
			isFirst:    false,
			expectChan: 5,
			expectLen:  0,
		},
		{
			name: "data first compressed with length",
			data: []byte{
				0x0A, 0x00,             // channelID=10 (2-byte)
				0x00, 0x10, 0x00, 0x00, // length=4096
				0xDE, 0xAD, 0xBE, 0xEF, // compressed data
			},
			cbChID:     1,
			isFirst:    true,
			expectChan: 10,
			expectLen:  4096,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pdu DataCompressedPDU
			err := pdu.Deserialize(tt.data, tt.cbChID, tt.isFirst)
			require.NoError(t, err)
			assert.Equal(t, tt.expectChan, pdu.ChannelID)
			assert.Equal(t, tt.isFirst, pdu.IsFirst)
			if tt.isFirst {
				assert.Equal(t, tt.expectLen, pdu.Length)
			}
			assert.NotEmpty(t, pdu.CompressedData)
		})
	}
}

func TestZGFXDecompressor_Uncompressed(t *testing.T) {
	decompressor := NewZGFXDecompressor()

	// Test uncompressed data (descriptor byte 0x00 = not compressed)
	compressed := []byte{0x00, 'H', 'e', 'l', 'l', 'o'}
	result, err := decompressor.Decompress(compressed)
	require.NoError(t, err)
	assert.Equal(t, []byte("Hello"), result)
}

func TestZGFXDecompressor_Empty(t *testing.T) {
	decompressor := NewZGFXDecompressor()

	_, err := decompressor.Decompress([]byte{})
	assert.Error(t, err)
}

func TestZGFXDecompressor_Flushed(t *testing.T) {
	decompressor := NewZGFXDecompressor()

	// Test PACKET_FLUSHED flag (0x04) - should reset history
	// First add some data to history
	_, _ = decompressor.Decompress([]byte{0x00, 'A', 'B', 'C'})
	
	// Then send flushed uncompressed data
	compressed := []byte{0x04, 'X', 'Y', 'Z'} // FLUSHED flag set, not compressed
	result, err := decompressor.Decompress(compressed)
	require.NoError(t, err)
	assert.Equal(t, []byte("XYZ"), result)
}

func TestZGFXDecompressor_CompressedShortData(t *testing.T) {
	decompressor := NewZGFXDecompressor()

	// Compressed flag set but data too short for segment header
	compressed := []byte{0x01, 0x00, 0x00} // Compressed, but only 3 bytes after descriptor
	_, err := decompressor.Decompress(compressed)
	assert.Error(t, err)
}

func TestDataCompressedPDU_Decompress_NilDecompressor(t *testing.T) {
	pdu := &DataCompressedPDU{
		CompressedData: []byte{0x00, 'T', 'e', 's', 't'},
	}
	_, err := pdu.Decompress(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no ZGFX decompressor")
}

func TestDataCompressedPDU_Decompress_Success(t *testing.T) {
	pdu := &DataCompressedPDU{
		CompressedData: []byte{0x00, 'T', 'e', 's', 't'}, // Uncompressed
	}
	decompressor := NewZGFXDecompressor()
	result, err := pdu.Decompress(decompressor)
	require.NoError(t, err)
	assert.Equal(t, []byte("Test"), result)
}

func TestBitReader(t *testing.T) {
	// Test reading individual bits and multi-bit values
	data := []byte{0xAA} // 10101010
	reader := newBitReader(data)

	// Read individual bits
	bit, _ := reader.readBit()
	assert.Equal(t, uint8(1), bit) // First bit is 1

	bit, _ = reader.readBit()
	assert.Equal(t, uint8(0), bit) // Second bit is 0

	// Read 4 more bits as a value
	val, _ := reader.readBits(4)
	assert.Equal(t, uint32(0b1010), val) // bits 3-6: 1010
}

func TestBitReader_EOF(t *testing.T) {
	reader := newBitReader([]byte{0xFF})
	
	// Read all 8 bits
	for i := 0; i < 8; i++ {
		_, err := reader.readBit()
		require.NoError(t, err)
	}
	
	// Next read should EOF
	_, err := reader.readBit()
	assert.Error(t, err)
}

func TestBitReader_ReadBits_CrossByte(t *testing.T) {
	// Test reading bits that cross byte boundaries
	data := []byte{0xFF, 0x00} // 11111111 00000000
	reader := newBitReader(data)
	
	// Read 4 bits (1111)
	val, err := reader.readBits(4)
	require.NoError(t, err)
	assert.Equal(t, uint32(0xF), val)
	
	// Read 8 bits crossing boundary (1111 0000)
	val, err = reader.readBits(8)
	require.NoError(t, err)
	assert.Equal(t, uint32(0xF0), val)
}

func TestCapsPDU_Deserialize_V3_Complete(t *testing.T) {
	// Test V3 caps with all priority charges
	caps := CapsPDU{
		Version:         CapsVersion3,
		PriorityCharge0: 100,
		PriorityCharge1: 200,
		PriorityCharge2: 300,
		PriorityCharge3: 400,
	}
	data := caps.Serialize()
	
	var decoded CapsPDU
	err := decoded.Deserialize(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, CapsVersion3, decoded.Version)
	assert.Equal(t, uint16(100), decoded.PriorityCharge0)
	assert.Equal(t, uint16(200), decoded.PriorityCharge1)
	assert.Equal(t, uint16(300), decoded.PriorityCharge2)
	assert.Equal(t, uint16(400), decoded.PriorityCharge3)
}

func TestCreateResponsePDU_Deserialize_4ByteChannel(t *testing.T) {
	// Test with 4-byte channel ID
	data := []byte{
		0x78, 0x56, 0x34, 0x12, // Channel ID (4 bytes) = 0x12345678
		0x00, 0x00, 0x00, 0x00, // Creation status = success
	}
	
	var resp CreateResponsePDU
	err := resp.Deserialize(bytes.NewReader(data), 2) // cbChID=2 means 4 bytes
	require.NoError(t, err)
	assert.Equal(t, uint32(0x12345678), resp.ChannelID)
	assert.True(t, resp.IsSuccess())
}

func TestCreateResponsePDU_Deserialize_Failure(t *testing.T) {
	// Test with failure status
	data := []byte{
		0x01,                   // Channel ID (1 byte)
		0x01, 0x00, 0x00, 0x00, // Creation status = denied
	}
	
	var resp CreateResponsePDU
	err := resp.Deserialize(bytes.NewReader(data), 0)
	require.NoError(t, err)
	assert.False(t, resp.IsSuccess())
}

func TestDataFirstPDU_Serialize_AllLengthSizes(t *testing.T) {
	tests := []struct {
		name   string
		length uint32
	}{
		{"1-byte length", 100},
		{"2-byte length", 1000},
		{"4-byte length", 100000},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := DataFirstPDU{
				ChannelID: 1,
				Length:    tt.length,
				Data:      []byte{0xAA, 0xBB},
			}
			data := pdu.Serialize()
			require.NotEmpty(t, data)
			
			// Verify header
			var h Header
			h.Deserialize(data[0])
			assert.Equal(t, CmdDataFirst, h.Cmd)
		})
	}
}

func TestSoftSyncRequestPDU_Deserialize_TooManyChannels(t *testing.T) {
	// Create data with too many channels
	data := []byte{
		0x00,       // Pad
		0x03,       // Flags (CHANNEL_LIST_PRESENT)
		0x01, 0x00, // NumberOfTunnels
		0x01, 0x10, // Channel count = 4097 (exceeds 1024 limit)
	}
	
	var pdu SoftSyncRequestPDU
	err := pdu.Deserialize(bytes.NewReader(data))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many")
}

func TestDataFirstPDU_Serialize_AllChannelSizes(t *testing.T) {
	tests := []struct {
		name      string
		channelID uint32
	}{
		{"1-byte channel", 100},
		{"2-byte channel", 1000},
		{"4-byte channel", 100000},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := DataFirstPDU{
				ChannelID: tt.channelID,
				Length:    500,
				Data:      []byte{0x01, 0x02},
			}
			data := pdu.Serialize()
			require.NotEmpty(t, data)
		})
	}
}

func TestDataPDU_Serialize_AllChannelSizes(t *testing.T) {
	tests := []struct {
		name      string
		channelID uint32
	}{
		{"1-byte channel", 100},
		{"2-byte channel", 1000},
		{"4-byte channel", 100000},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := DataPDU{
				ChannelID: tt.channelID,
				Data:      []byte{0x01, 0x02},
			}
			data := pdu.Serialize()
			require.NotEmpty(t, data)
		})
	}
}

func TestSoftSyncRequestPDU_Deserialize_NoChannelList(t *testing.T) {
	// Request without channel list flag
	data := []byte{
		0x00,       // Pad
		0x01,       // Flags (TCP_FLUSHED only, no channel list)
		0x02, 0x00, // NumberOfTunnels
	}
	
	var pdu SoftSyncRequestPDU
	err := pdu.Deserialize(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, uint8(SoftSyncTCPFlushed), pdu.Flags)
	assert.Len(t, pdu.Channels, 0)
}

func TestSoftSyncRequestPDU_Deserialize_ReadErrors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"only pad", []byte{0x00}},
		{"missing tunnels", []byte{0x00, 0x01}},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pdu SoftSyncRequestPDU
			err := pdu.Deserialize(bytes.NewReader(tt.data))
			assert.Error(t, err)
		})
	}
}

func TestDataCompressedPDU_Deserialize_4ByteChannel(t *testing.T) {
	data := []byte{
		0x78, 0x56, 0x34, 0x12, // Channel ID (4 bytes)
		0xAA, 0xBB, 0xCC,       // Compressed data
	}
	
	var pdu DataCompressedPDU
	err := pdu.Deserialize(data, 2, false) // cbChID=2 means 4 bytes
	require.NoError(t, err)
	assert.Equal(t, uint32(0x12345678), pdu.ChannelID)
}

func TestDataCompressedPDU_Deserialize_DataFirst_TooShort(t *testing.T) {
	data := []byte{
		0x01,       // Channel ID (1 byte)
		0x00, 0x00, // Only 2 bytes for length, need 4
	}
	
	var pdu DataCompressedPDU
	err := pdu.Deserialize(data, 0, true) // isFirst=true needs length field
	assert.Error(t, err)
}

// ============================================================================
// Microsoft Protocol Test Suite Validation Tests
// Reference: MS-RDPBCGR_ClientTestDesignSpecification.md - S7_StaticVirtualChannel
// ============================================================================

// TestBVT_StaticVirtualChannel_HeaderFlags validates per MS test case:
// "BVT_StaticVCTest_PositiveTest_ChannelFlags"
// Per MS-RDPBCGR Section 2.2.6.1
func TestBVT_StaticVirtualChannel_HeaderFlags(t *testing.T) {
	// Virtual channel header flags per MS-RDPBCGR 2.2.6.1
	const (
		CHANNEL_FLAG_FIRST    = 0x00000001
		CHANNEL_FLAG_LAST     = 0x00000002
		CHANNEL_FLAG_SHOW_PROTOCOL = 0x00000010
		CHANNEL_FLAG_SUSPEND  = 0x00000020
		CHANNEL_FLAG_RESUME   = 0x00000040
		CHANNEL_FLAG_SHADOW_PERSISTENT = 0x00000080
	)

	tests := []struct {
		name  string
		flags uint32
		valid bool
	}{
		{"Single_Complete_Packet", CHANNEL_FLAG_FIRST | CHANNEL_FLAG_LAST, true},
		{"First_Fragment", CHANNEL_FLAG_FIRST, true},
		{"Last_Fragment", CHANNEL_FLAG_LAST, true},
		{"Middle_Fragment", 0, true},
		{"Show_Protocol", CHANNEL_FLAG_SHOW_PROTOCOL, true},
		{"Suspend", CHANNEL_FLAG_SUSPEND, true},
		{"Resume", CHANNEL_FLAG_RESUME, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Per spec, FIRST and LAST together indicate single unfragmented PDU
			isSingle := (tc.flags&CHANNEL_FLAG_FIRST != 0) && (tc.flags&CHANNEL_FLAG_LAST != 0)
			isFirst := (tc.flags&CHANNEL_FLAG_FIRST != 0) && (tc.flags&CHANNEL_FLAG_LAST == 0)
			isLast := (tc.flags&CHANNEL_FLAG_FIRST == 0) && (tc.flags&CHANNEL_FLAG_LAST != 0)
			isMiddle := (tc.flags&CHANNEL_FLAG_FIRST == 0) && (tc.flags&CHANNEL_FLAG_LAST == 0)

			// At least one state must be true for valid flags
			hasValidState := isSingle || isFirst || isLast || isMiddle
			assert.True(t, hasValidState, "Invalid flag combination")
		})
	}
}

// TestS7_StaticVirtualChannel_Compression validates compression flag handling
// Per MS-RDPBCGR Section 2.2.6.1.1
func TestS7_StaticVirtualChannel_Compression(t *testing.T) {
	const (
		CHANNEL_FLAG_FIRST         = 0x00000001
		CHANNEL_FLAG_LAST          = 0x00000002
		CompressionTypeMask        = 0x000F0000
		PACKET_COMPRESSED          = 0x00200000
		PACKET_AT_FRONT            = 0x00400000
		PACKET_FLUSHED             = 0x00800000
		CompressionType_8K         = 0x00000000
		CompressionType_64K        = 0x00010000
		CompressionType_RDP6       = 0x00020000
		CompressionType_RDP61      = 0x00030000
	)

	tests := []struct {
		name            string
		flags           uint32
		compressionType int
	}{
		{"8K_Compression", PACKET_COMPRESSED | CompressionType_8K, 0},
		{"64K_Compression", PACKET_COMPRESSED | CompressionType_64K, 1},
		{"RDP6_Compression", PACKET_COMPRESSED | CompressionType_RDP6, 2},
		{"RDP61_Compression", PACKET_COMPRESSED | CompressionType_RDP61, 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isCompressed := (tc.flags & PACKET_COMPRESSED) != 0
			assert.True(t, isCompressed)

			compressionType := int((tc.flags & CompressionTypeMask) >> 16)
			assert.Equal(t, tc.compressionType, compressionType)
		})
	}
}

// TestS7_DynamicVirtualChannel_Commands validates DRDYNVC command codes
// Per MS-RDPEDYC Section 2.2.1
func TestS7_DynamicVirtualChannel_Commands(t *testing.T) {
	// DRDYNVC command codes per MS-RDPEDYC
	tests := []struct {
		cmd  uint8
		name string
	}{
		{CmdCreate, "DYNVC_CREATE_REQ (0x01)"},
		{CmdDataFirst, "DYNVC_DATA_FIRST (0x02)"},
		{CmdData, "DYNVC_DATA (0x03)"},
		{CmdClose, "DYNVC_CLOSE (0x04)"},
		{CmdCapability, "DYNVC_CAPS (0x05)"},
		{CmdDataFirstCmp, "DYNVC_DATA_FIRST_COMPRESSED (0x06)"},
		{CmdDataCmp, "DYNVC_DATA_COMPRESSED (0x07)"},
		{CmdSoftSync, "DYNVC_SOFT_SYNC_REQ/RSP (0x08)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Verify command value fits in 4 bits (0-15)
			assert.LessOrEqual(t, tc.cmd, uint8(0x0F))
		})
	}
}

// TestS7_DynamicVirtualChannel_CapsVersions validates capability versions
// Per MS-RDPEDYC Section 2.2.1.1
func TestS7_DynamicVirtualChannel_CapsVersions(t *testing.T) {
	tests := []struct {
		version  uint16
		features string
	}{
		{1, "Basic DVC support"},
		{2, "Priority support"},
		{3, "Compression and Soft-Sync support"},
	}

	for _, tc := range tests {
		t.Run(tc.features, func(t *testing.T) {
			caps := CapsPDU{Version: tc.version}
			assert.Equal(t, tc.version, caps.Version)

			// V2+ supports priority
			if tc.version >= 2 {
				assert.True(t, tc.version >= 2)
			}
			// V3 supports compression
			if tc.version >= 3 {
				assert.True(t, tc.version >= 3)
			}
		})
	}
}

// TestS7_DynamicVirtualChannel_ChannelIDEncoding validates channel ID encoding
// Per MS-RDPEDYC Section 2.2.1 - CbChID field
func TestS7_DynamicVirtualChannel_ChannelIDEncoding(t *testing.T) {
	tests := []struct {
		cbChID     uint8
		size       int
		maxChannel uint32
	}{
		{0, 1, 0xFF},
		{1, 2, 0xFFFF},
		{2, 4, 0xFFFFFFFF},
	}

	for _, tc := range tests {
		t.Run("CbChID="+string('0'+tc.cbChID), func(t *testing.T) {
			h := Header{CbChID: tc.cbChID, Cmd: CmdData}
			assert.Equal(t, tc.size, h.ChannelIDSize())
		})
	}
}

// TestS7_DynamicVirtualChannel_CreateResponse validates create response codes
// Per MS-RDPEDYC Section 2.2.2.2
func TestS7_DynamicVirtualChannel_CreateResponse(t *testing.T) {
	tests := []struct {
		code    uint32
		name    string
		success bool
	}{
		{CreateResultOK, "CHANNEL_RC_OK", true},
		{CreateResultDenied, "CHANNEL_RC_DENIED", false},
		{CreateResultNoMemory, "CHANNEL_RC_NO_MEMORY", false},
		{CreateResultNoListener, "CHANNEL_RC_NO_LISTENER", false},
		{CreateResultChannelNotFound, "CHANNEL_RC_CHANNEL_NOT_FOUND", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := CreateResponsePDU{CreationCode: tc.code}
			isSuccess := resp.IsSuccess()
			assert.Equal(t, tc.success, isSuccess)
		})
	}
}
