package rdp

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRailState_Constants(t *testing.T) {
	assert.Equal(t, RailState(0), RailStateUninitialized)
	assert.Equal(t, RailState(1), RailStateInitializing)
	assert.Equal(t, RailState(2), RailStateSyncDesktop)
	assert.Equal(t, RailState(3), RailStateWaitForData)
	assert.Equal(t, RailState(4), RailStateExecuteApp)
}

func TestRailOrder_Constants(t *testing.T) {
	tests := []struct {
		name     string
		order    RailOrder
		expected uint16
	}{
		{"RailOrderExec", RailOrderExec, 0x0001},
		{"RailOrderActivate", RailOrderActivate, 0x0002},
		{"RailOrderSysParam", RailOrderSysParam, 0x0003},
		{"RailOrderSysCommand", RailOrderSysCommand, 0x0004},
		{"RailOrderHandshake", RailOrderHandshake, 0x0005},
		{"RailOrderNotifyEvent", RailOrderNotifyEvent, 0x0006},
		{"RailOrderWindowMove", RailOrderWindowMove, 0x0008},
		{"RailOrderLocalMoveSize", RailOrderLocalMoveSize, 0x0009},
		{"RailOrderMinMaxInfo", RailOrderMinMaxInfo, 0x000a},
		{"RailOrderClientStatus", RailOrderClientStatus, 0x000b},
		{"RailOrderSysMenu", RailOrderSysMenu, 0x000c},
		{"RailOrderLangBarInfo", RailOrderLangBarInfo, 0x000d},
		{"RailOrderExecResult", RailOrderExecResult, 0x0080},
		{"RailOrderGetAppIDReq", RailOrderGetAppIDReq, 0x000E},
		{"RailOrderAppIDResp", RailOrderAppIDResp, 0x000F},
		{"RailOrderTaskBarInfo", RailOrderTaskBarInfo, 0x0010},
		{"RailOrderLanguageIMEInfo", RailOrderLanguageIMEInfo, 0x0011},
		{"RailOrderCompartmentInfo", RailOrderCompartmentInfo, 0x0012},
		{"RailOrderHandshakeEx", RailOrderHandshakeEx, 0x0013},
		{"RailOrderZOrderSync", RailOrderZOrderSync, 0x0014},
		{"RailOrderCloak", RailOrderCloak, 0x0015},
		{"RailOrderPowerDisplayRequest", RailOrderPowerDisplayRequest, 0x0016},
		{"RailOrderSnapArrange", RailOrderSnapArrange, 0x0017},
		{"RailOrderGetAppIDRespEx", RailOrderGetAppIDRespEx, 0x0018},
		{"RailOrderTextScaleInfo", RailOrderTextScaleInfo, 0x0019},
		{"RailOrderCaretBlinkInfo", RailOrderCaretBlinkInfo, 0x001A},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, uint16(tt.order))
		})
	}
}

func TestRailPDUHeader_Serialize(t *testing.T) {
	header := RailPDUHeader{
		OrderType:   RailOrderHandshake,
		OrderLength: 16,
	}

	result := header.Serialize()
	require.Len(t, result, 4)

	// Check OrderType (2 bytes, little endian)
	assert.Equal(t, uint16(RailOrderHandshake), binary.LittleEndian.Uint16(result[0:2]))
	// Check OrderLength (2 bytes, little endian)
	assert.Equal(t, uint16(16), binary.LittleEndian.Uint16(result[2:4]))
}

func TestRailPDUHeader_Deserialize(t *testing.T) {
	tests := []struct {
		name              string
		data              []byte
		expectedOrderType RailOrder
		expectedLength    uint16
		expectError       bool
	}{
		{
			name:              "valid handshake header",
			data:              []byte{0x05, 0x00, 0x10, 0x00},
			expectedOrderType: RailOrderHandshake,
			expectedLength:    16,
			expectError:       false,
		},
		{
			name:              "valid exec header",
			data:              []byte{0x01, 0x00, 0x20, 0x00},
			expectedOrderType: RailOrderExec,
			expectedLength:    32,
			expectError:       false,
		},
		{
			name:              "exec result header",
			data:              []byte{0x80, 0x00, 0x40, 0x00},
			expectedOrderType: RailOrderExecResult,
			expectedLength:    64,
			expectError:       false,
		},
		{
			name:        "truncated data",
			data:        []byte{0x05, 0x00},
			expectError: true,
		},
		{
			name:        "empty data",
			data:        []byte{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &RailPDUHeader{}
			err := header.Deserialize(bytes.NewReader(tt.data))

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedOrderType, header.OrderType)
				assert.Equal(t, tt.expectedLength, header.OrderLength)
			}
		})
	}
}

func TestRailPDUHandshake_Serialize(t *testing.T) {
	handshake := RailPDUHandshake{
		buildNumber: 0x00001DB0,
	}

	result := handshake.Serialize()
	require.Len(t, result, 4)

	// Check buildNumber (4 bytes, little endian)
	assert.Equal(t, uint32(0x00001DB0), binary.LittleEndian.Uint32(result[0:4]))
}

func TestRailPDUHandshake_Deserialize(t *testing.T) {
	tests := []struct {
		name                string
		data                []byte
		expectedBuildNumber uint32
		expectError         bool
	}{
		{
			name:                "valid build number",
			data:                []byte{0xB0, 0x1D, 0x00, 0x00},
			expectedBuildNumber: 0x00001DB0,
			expectError:         false,
		},
		{
			name:                "max build number",
			data:                []byte{0xFF, 0xFF, 0xFF, 0xFF},
			expectedBuildNumber: 0xFFFFFFFF,
			expectError:         false,
		},
		{
			name:        "truncated data",
			data:        []byte{0xB0, 0x1D},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handshake := &RailPDUHandshake{}
			err := handshake.Deserialize(bytes.NewReader(tt.data))

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBuildNumber, handshake.buildNumber)
			}
		})
	}
}

func TestRailPDUClientInfo_Serialize(t *testing.T) {
	clientInfo := RailPDUClientInfo{
		Flags: 0x12345678,
	}

	result := clientInfo.Serialize()
	require.Len(t, result, 4)

	// Check Flags (4 bytes, little endian)
	assert.Equal(t, uint32(0x12345678), binary.LittleEndian.Uint32(result[0:4]))
}

func TestRailPDUClientSystemParamUpdate_Serialize(t *testing.T) {
	update := RailPDUClientSystemParamUpdate{
		SystemParam: 0xABCDEF01,
		Body:        0x42,
	}

	result := update.Serialize()
	require.Len(t, result, 5)

	// Check SystemParam (4 bytes, little endian)
	assert.Equal(t, uint32(0xABCDEF01), binary.LittleEndian.Uint32(result[0:4]))
	// Check Body (1 byte)
	assert.Equal(t, uint8(0x42), result[4])
}

func TestRailPDUSystemParameters_Deserialize(t *testing.T) {
	tests := []struct {
		name               string
		data               []byte
		expectedSysParam   uint32
		expectedBody       uint8
		expectError        bool
	}{
		{
			name:             "valid system parameters",
			data:             []byte{0x01, 0xEF, 0xCD, 0xAB, 0x42},
			expectedSysParam: 0xABCDEF01,
			expectedBody:     0x42,
			expectError:      false,
		},
		{
			name:        "truncated at body",
			data:        []byte{0x01, 0xEF, 0xCD, 0xAB},
			expectError: true,
		},
		{
			name:        "truncated at system param",
			data:        []byte{0x01, 0xEF},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sysParams := &RailPDUSystemParameters{}
			err := sysParams.Deserialize(bytes.NewReader(tt.data))

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedSysParam, sysParams.SystemParameter)
				assert.Equal(t, tt.expectedBody, sysParams.Body)
			}
		})
	}
}

func TestRailPDUExecResult_Deserialize(t *testing.T) {
	// Build a valid exec result packet
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint16(0x0001))      // Flags
	binary.Write(buf, binary.LittleEndian, uint16(0x0000))      // ExecResult (success)
	binary.Write(buf, binary.LittleEndian, uint32(0x00000000))  // RawResult
	binary.Write(buf, binary.LittleEndian, uint16(0x0000))      // Padding
	binary.Write(buf, binary.LittleEndian, uint16(16))          // ExeOrFileLength
	buf.Write([]byte("notepad.exe\x00\x00\x00\x00\x00"))        // ExeOrFile (padded to 16)

	execResult := &RailPDUExecResult{}
	err := execResult.Deserialize(bytes.NewReader(buf.Bytes()))

	require.NoError(t, err)
	assert.Equal(t, uint16(0x0001), execResult.Flags)
	assert.Equal(t, uint16(0x0000), execResult.ExecResult)
	assert.Equal(t, uint32(0x00000000), execResult.RawResult)
	assert.Equal(t, "notepad.exe\x00\x00\x00\x00\x00", execResult.ExeOrFile)
}

func TestNewRailHandshakePDU(t *testing.T) {
	pdu := NewRailHandshakePDU()

	require.NotNil(t, pdu)
	assert.Equal(t, ChannelFlagFirst|ChannelFlagLast, pdu.channelHeader.Flags)
	assert.Equal(t, RailOrderHandshake, pdu.header.OrderType)
	require.NotNil(t, pdu.RailPDUHandshake)
	assert.Equal(t, uint32(0x00001DB0), pdu.RailPDUHandshake.buildNumber)
}

func TestNewRailClientInfoPDU(t *testing.T) {
	pdu := NewRailClientInfoPDU()

	require.NotNil(t, pdu)
	assert.Equal(t, ChannelFlagFirst|ChannelFlagLast, pdu.channelHeader.Flags)
	assert.Equal(t, RailOrderClientStatus, pdu.header.OrderType)
	require.NotNil(t, pdu.RailPDUClientInfo)
	assert.Equal(t, uint32(0), pdu.RailPDUClientInfo.Flags)
}

func TestNewRailPDUClientSystemParamUpdate(t *testing.T) {
	pdu := NewRailPDUClientSystemParamUpdate(0x12345678, 0xAB)

	require.NotNil(t, pdu)
	assert.Equal(t, ChannelFlagFirst|ChannelFlagLast, pdu.channelHeader.Flags)
	assert.Equal(t, RailOrderSysParam, pdu.header.OrderType)
	require.NotNil(t, pdu.RailPDUClientSystemParamUpdate)
	assert.Equal(t, uint32(0x12345678), pdu.RailPDUClientSystemParamUpdate.SystemParam)
	assert.Equal(t, uint8(0xAB), pdu.RailPDUClientSystemParamUpdate.Body)
}

func TestNewRailClientExecutePDU(t *testing.T) {
	pdu := NewRailClientExecutePDU("notepad.exe", "C:\\Windows", "--help")

	require.NotNil(t, pdu)
	assert.Equal(t, ChannelFlagFirst|ChannelFlagLast, pdu.channelHeader.Flags)
	assert.Equal(t, RailOrderExec, pdu.header.OrderType)
	require.NotNil(t, pdu.RailPDUClientExecute)
	assert.Equal(t, "notepad.exe", pdu.RailPDUClientExecute.ExeOrFile)
	assert.Equal(t, "C:\\Windows", pdu.RailPDUClientExecute.WorkingDir)
	assert.Equal(t, "--help", pdu.RailPDUClientExecute.Arguments)
}

func TestRailPDU_Serialize_Handshake(t *testing.T) {
	pdu := NewRailHandshakePDU()
	result := pdu.Serialize()

	// Should have channel header (8 bytes) + rail header (4 bytes) + handshake data (4 bytes)
	require.GreaterOrEqual(t, len(result), 16)

	// Verify the order length was calculated
	assert.Greater(t, pdu.header.OrderLength, uint16(0))
}

func TestRailPDU_Serialize_Exec(t *testing.T) {
	pdu := NewRailClientExecutePDU("calc.exe", "", "")
	result := pdu.Serialize()

	// Should have channel header + rail header + exec data
	require.Greater(t, len(result), 12)
}

func TestRailPDU_Serialize_SysParam(t *testing.T) {
	pdu := NewRailPDUClientSystemParamUpdate(0x01, 0x02)
	result := pdu.Serialize()

	// Should have channel header + rail header + sysparam data (5 bytes)
	require.Greater(t, len(result), 12)
}

func TestRailPDU_Deserialize_Handshake(t *testing.T) {
	// Build a handshake PDU
	buf := new(bytes.Buffer)

	// Channel header (8 bytes)
	binary.Write(buf, binary.LittleEndian, uint32(8))  // length
	binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))

	// Rail header (4 bytes)
	binary.Write(buf, binary.LittleEndian, uint16(RailOrderHandshake))
	binary.Write(buf, binary.LittleEndian, uint16(16))  // OrderLength

	// Handshake data (4 bytes)
	binary.Write(buf, binary.LittleEndian, uint32(0x00001DB0))

	pdu := &RailPDU{}
	err := pdu.Deserialize(bytes.NewReader(buf.Bytes()))

	require.NoError(t, err)
	assert.Equal(t, RailOrderHandshake, pdu.header.OrderType)
	require.NotNil(t, pdu.RailPDUHandshake)
	assert.Equal(t, uint32(0x00001DB0), pdu.RailPDUHandshake.buildNumber)
}

func TestRailPDU_Deserialize_SysParam(t *testing.T) {
	buf := new(bytes.Buffer)

	// Channel header
	binary.Write(buf, binary.LittleEndian, uint32(8))
	binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))

	// Rail header
	binary.Write(buf, binary.LittleEndian, uint16(RailOrderSysParam))
	binary.Write(buf, binary.LittleEndian, uint16(17))

	// SysParam data
	binary.Write(buf, binary.LittleEndian, uint32(0x12345678))
	binary.Write(buf, binary.LittleEndian, uint8(0x42))

	pdu := &RailPDU{}
	err := pdu.Deserialize(bytes.NewReader(buf.Bytes()))

	require.NoError(t, err)
	assert.Equal(t, RailOrderSysParam, pdu.header.OrderType)
	require.NotNil(t, pdu.RailPDUSystemParameters)
	assert.Equal(t, uint32(0x12345678), pdu.RailPDUSystemParameters.SystemParameter)
	assert.Equal(t, uint8(0x42), pdu.RailPDUSystemParameters.Body)
}

func TestRailPDU_Deserialize_ExecResult(t *testing.T) {
	buf := new(bytes.Buffer)

	// Channel header
	binary.Write(buf, binary.LittleEndian, uint32(8))
	binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))

	// Rail header
	binary.Write(buf, binary.LittleEndian, uint16(RailOrderExecResult))
	binary.Write(buf, binary.LittleEndian, uint16(32))

	// ExecResult data
	binary.Write(buf, binary.LittleEndian, uint16(0x0001))      // Flags
	binary.Write(buf, binary.LittleEndian, uint16(0x0000))      // ExecResult
	binary.Write(buf, binary.LittleEndian, uint32(0x00000000))  // RawResult
	binary.Write(buf, binary.LittleEndian, uint16(0x0000))      // Padding
	binary.Write(buf, binary.LittleEndian, uint16(4))           // ExeOrFileLength
	buf.Write([]byte("test"))                                   // ExeOrFile

	pdu := &RailPDU{}
	err := pdu.Deserialize(bytes.NewReader(buf.Bytes()))

	require.NoError(t, err)
	assert.Equal(t, RailOrderExecResult, pdu.header.OrderType)
	require.NotNil(t, pdu.RailPDUExecResult)
	assert.Equal(t, "test", pdu.RailPDUExecResult.ExeOrFile)
}

func TestRailPDU_Deserialize_UnknownOrderType(t *testing.T) {
	buf := new(bytes.Buffer)

	// Channel header
	binary.Write(buf, binary.LittleEndian, uint32(8))
	binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))

	// Rail header with unknown order type
	binary.Write(buf, binary.LittleEndian, uint16(0xFFFF))  // Unknown order type
	binary.Write(buf, binary.LittleEndian, uint16(12))

	pdu := &RailPDU{}
	err := pdu.Deserialize(bytes.NewReader(buf.Bytes()))

	// Should not error, just skip the unknown order
	require.NoError(t, err)
	assert.Equal(t, RailOrder(0xFFFF), pdu.header.OrderType)
}
