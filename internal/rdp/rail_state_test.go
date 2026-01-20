package rdp

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Client's handleRail with different states
func TestClient_handleRail_InitializingState(t *testing.T) {
	// When in initializing state and order is Handshake, handleRail should call railHandshake
	// Since we can't mock mcsLayer, we test that SysParam orders are handled first
	client := &Client{
		remoteApp: &RemoteApp{App: "test.exe"},
		railState: RailStateInitializing,
	}

	// Build a SysParam PDU (which returns early)
	buf := new(bytes.Buffer)
	// Channel header
	_ = binary.Write(buf, binary.LittleEndian, uint32(8))
	_ = binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))
	// Rail header (SysParam)
	_ = binary.Write(buf, binary.LittleEndian, uint16(RailOrderSysParam))
	_ = binary.Write(buf, binary.LittleEndian, uint16(17))
	// SysParam data
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x12345678))
	_ = binary.Write(buf, binary.LittleEndian, uint8(0x42))

	// SysParam should return early without calling railHandshake
	err := client.handleRail(buf)
	assert.NoError(t, err)
}

func TestClient_handleRail_ExecuteAppState(t *testing.T) {
	client := &Client{
		remoteApp: &RemoteApp{App: "test.exe"},
		railState: RailStateExecuteApp,
	}

	// Build an exec result PDU
	buf := new(bytes.Buffer)
	// Channel header
	_ = binary.Write(buf, binary.LittleEndian, uint32(8))
	_ = binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))
	// Rail header
	_ = binary.Write(buf, binary.LittleEndian, uint16(RailOrderExecResult))
	_ = binary.Write(buf, binary.LittleEndian, uint16(32))
	// ExecResult data
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0001))      // Flags
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000))      // ExecResult
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x00000000))  // RawResult
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0000))      // Padding
	_ = binary.Write(buf, binary.LittleEndian, uint16(4))           // ExeOrFileLength
	buf.Write([]byte("test"))                                   // ExeOrFile

	err := client.handleRail(buf)
	assert.NoError(t, err)
	assert.Equal(t, RailStateWaitForData, client.railState)
}

func TestClient_handleRail_WaitForDataState(t *testing.T) {
	client := &Client{
		remoteApp: &RemoteApp{App: "test.exe"},
		railState: RailStateWaitForData,
	}

	// Build a non-sysparam PDU that won't match any state handler
	buf := new(bytes.Buffer)
	// Channel header
	_ = binary.Write(buf, binary.LittleEndian, uint32(8))
	_ = binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))
	// Rail header with unknown order type
	_ = binary.Write(buf, binary.LittleEndian, uint16(RailOrderActivate))
	_ = binary.Write(buf, binary.LittleEndian, uint16(12))

	err := client.handleRail(buf)
	// Should not error, just return nil (no matching state handler)
	assert.NoError(t, err)
}

func TestClient_handleRail_DeserializeError(t *testing.T) {
	client := &Client{
		remoteApp: &RemoteApp{App: "test.exe"},
		railState: RailStateWaitForData,
	}

	// Truncated data
	err := client.handleRail(bytes.NewReader([]byte{0x01}))
	assert.Error(t, err)
}

func TestRailPDU_Serialize_AllTypes(t *testing.T) {
	tests := []struct {
		name string
		pdu  *RailPDU
	}{
		{
			name: "handshake",
			pdu:  NewRailHandshakePDU(),
		},
		{
			name: "sysparam",
			pdu:  NewRailPDUClientSystemParamUpdate(0x01, 0x02),
		},
		{
			name: "exec",
			pdu:  NewRailClientExecutePDU("app.exe", "C:\\", "args"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pdu.Serialize()
			require.NotEmpty(t, result)

			// Should have channel header (8 bytes) + rail header (4 bytes) + data
			require.GreaterOrEqual(t, len(result), 12)
		})
	}
}

func TestRailPDU_Deserialize_CompleteFlow(t *testing.T) {
	// Create a handshake PDU, serialize it, then deserialize
	original := NewRailHandshakePDU()
	serialized := original.Serialize()

	deserialized := &RailPDU{}
	err := deserialized.Deserialize(bytes.NewReader(serialized))

	require.NoError(t, err)
	assert.Equal(t, RailOrderHandshake, deserialized.header.OrderType)
	require.NotNil(t, deserialized.RailPDUHandshake)
}

func TestChannelPDUHeader_FlagCombinations(t *testing.T) {
	flagCombinations := []ChannelFlag{
		ChannelFlagFirst | ChannelFlagLast,
		ChannelFlagFirst | ChannelFlagCompressed,
		ChannelFlagLast | ChannelFlagFlushed,
		ChannelFlagShowProtocol | ChannelFlagSuspend | ChannelFlagResume,
		ChannelFlagFirst | ChannelFlagLast | ChannelFlagCompressed | ChannelFlagAtFront | ChannelFlagFlushed,
	}

	for _, flags := range flagCombinations {
		t.Run("flags", func(t *testing.T) {
			header := ChannelPDUHeader{Flags: flags}
			serialized := header.Serialize()

			deserialized := &ChannelPDUHeader{}
			err := deserialized.Deserialize(bytes.NewReader(serialized))

			require.NoError(t, err)
			assert.Equal(t, flags, deserialized.Flags)
		})
	}
}

// Test error paths in deserialization
func TestChannelPDUHeader_Deserialize_PartialData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"1 byte", []byte{0x08}},
		{"2 bytes", []byte{0x08, 0x00}},
		{"3 bytes", []byte{0x08, 0x00, 0x00}},
		{"4 bytes only", []byte{0x08, 0x00, 0x00, 0x00}},
		{"5 bytes", []byte{0x08, 0x00, 0x00, 0x00, 0x03}},
		{"6 bytes", []byte{0x08, 0x00, 0x00, 0x00, 0x03, 0x00}},
		{"7 bytes", []byte{0x08, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &ChannelPDUHeader{}
			err := header.Deserialize(bytes.NewReader(tt.data))
			
			if len(tt.data) < 8 {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test RailPDUHeader serialization edge cases
func TestRailPDUHeader_EdgeCases(t *testing.T) {
	tests := []struct {
		orderType   RailOrder
		orderLength uint16
	}{
		{0, 0},
		{RailOrderExec, 0},
		{0xFFFF, 0xFFFF},
		{RailOrderHandshake, 1},
	}

	for _, tt := range tests {
		header := RailPDUHeader{
			OrderType:   tt.orderType,
			OrderLength: tt.orderLength,
		}

		serialized := header.Serialize()
		deserialized := &RailPDUHeader{}
		err := deserialized.Deserialize(bytes.NewReader(serialized))

		require.NoError(t, err)
		assert.Equal(t, tt.orderType, deserialized.OrderType)
		assert.Equal(t, tt.orderLength, deserialized.OrderLength)
	}
}
