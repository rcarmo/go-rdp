package rdp

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_handleRail_NilRemoteApp(t *testing.T) {
	client := &Client{
		remoteApp: nil,
	}

	// Build some dummy wire data
	wire := bytes.NewReader([]byte{})

	err := client.handleRail(wire)
	assert.NoError(t, err)
}

func TestClient_handleRail_SysParam(t *testing.T) {
	client := &Client{
		remoteApp: &RemoteApp{},
		railState: RailStateWaitForData,
	}

	// Build a SysParam PDU
	buf := new(bytes.Buffer)
	// Channel header
	_ = binary.Write(buf, binary.LittleEndian, uint32(8))
	_ = binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))
	// Rail header
	_ = binary.Write(buf, binary.LittleEndian, uint16(RailOrderSysParam))
	_ = binary.Write(buf, binary.LittleEndian, uint16(17))
	// SysParam data
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x12345678))
	_ = binary.Write(buf, binary.LittleEndian, uint8(0x42))

	err := client.handleRail(buf)
	assert.NoError(t, err)
}

func TestClient_railReceiveRemoteAppStatus(t *testing.T) {
	client := &Client{
		railState: RailStateExecuteApp,
	}

	err := client.railReceiveRemoteAppStatus(nil)
	assert.NoError(t, err)
	assert.Equal(t, RailStateWaitForData, client.railState)
}

func TestRailPDU_SerializeDeserializeRoundTrip_Handshake(t *testing.T) {
	original := NewRailHandshakePDU()
	serialized := original.Serialize()

	// Deserialize
	deserialized := &RailPDU{}
	err := deserialized.Deserialize(bytes.NewReader(serialized))

	require.NoError(t, err)
	assert.Equal(t, original.header.OrderType, deserialized.header.OrderType)
	require.NotNil(t, deserialized.RailPDUHandshake)
	assert.Equal(t, original.RailPDUHandshake.buildNumber, deserialized.RailPDUHandshake.buildNumber)
}

func TestRailPDU_SerializeDeserializeRoundTrip_SysParam(t *testing.T) {
	original := NewRailPDUClientSystemParamUpdate(0xDEADBEEF, 0x42)
	serialized := original.Serialize()

	// Deserialize
	deserialized := &RailPDU{}
	err := deserialized.Deserialize(bytes.NewReader(serialized))

	require.NoError(t, err)
	assert.Equal(t, RailOrderSysParam, deserialized.header.OrderType)
	require.NotNil(t, deserialized.RailPDUSystemParameters)
}

func TestRailPDUClientExecute_Serialize_EmptyFields(t *testing.T) {
	exec := &RailPDUClientExecute{
		Flags:      0,
		ExeOrFile:  "",
		WorkingDir: "",
		Arguments:  "",
	}

	result := exec.Serialize()
	// Should have at least 8 bytes for the length fields
	require.GreaterOrEqual(t, len(result), 8)
}

func TestRailPDUClientExecute_Serialize_UTF16(t *testing.T) {
	exec := &RailPDUClientExecute{
		Flags:      0x0001,
		ExeOrFile:  "notepad.exe",
		WorkingDir: "C:\\",
		Arguments:  "test",
	}

	result := exec.Serialize()
	
	// Check flags
	flags := binary.LittleEndian.Uint16(result[0:2])
	assert.Equal(t, uint16(0x0001), flags)
}

func TestRailPDUExecResult_DeserializeError(t *testing.T) {
	// Truncated data should error
	execResult := &RailPDUExecResult{}
	err := execResult.Deserialize(bytes.NewReader([]byte{0x01, 0x00}))
	require.Error(t, err)
}

func TestRailPDU_Deserialize_ChannelHeaderError(t *testing.T) {
	// Empty data should fail at channel header
	pdu := &RailPDU{}
	err := pdu.Deserialize(bytes.NewReader([]byte{}))
	require.Error(t, err)
}

func TestRailPDU_Deserialize_RailHeaderError(t *testing.T) {
	// Only channel header, no rail header
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, uint32(8))
	_ = binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))
	// Missing rail header

	pdu := &RailPDU{}
	err := pdu.Deserialize(buf)
	require.Error(t, err)
}

func TestRailPDUHeader_SerializeDeserializeRoundTrip(t *testing.T) {
	tests := []struct {
		orderType   RailOrder
		orderLength uint16
	}{
		{RailOrderHandshake, 16},
		{RailOrderExec, 100},
		{RailOrderSysParam, 20},
		{RailOrderExecResult, 50},
		{RailOrderClientStatus, 12},
	}

	for _, tt := range tests {
		original := RailPDUHeader{
			OrderType:   tt.orderType,
			OrderLength: tt.orderLength,
		}

		serialized := original.Serialize()
		require.Len(t, serialized, 4)

		deserialized := &RailPDUHeader{}
		err := deserialized.Deserialize(bytes.NewReader(serialized))
		require.NoError(t, err)

		assert.Equal(t, original.OrderType, deserialized.OrderType)
		assert.Equal(t, original.OrderLength, deserialized.OrderLength)
	}
}

func TestRailPDUHandshake_SerializeDeserializeRoundTrip(t *testing.T) {
	buildNumbers := []uint32{0, 1, 0x00001DB0, 0xFFFFFFFF, 0x12345678}

	for _, bn := range buildNumbers {
		original := RailPDUHandshake{buildNumber: bn}
		serialized := original.Serialize()

		deserialized := &RailPDUHandshake{}
		err := deserialized.Deserialize(bytes.NewReader(serialized))
		require.NoError(t, err)

		assert.Equal(t, original.buildNumber, deserialized.buildNumber)
	}
}

func TestRailPDUSystemParameters_DeserializeError(t *testing.T) {
	// Truncated data
	sysParams := &RailPDUSystemParameters{}
	err := sysParams.Deserialize(bytes.NewReader([]byte{0x01}))
	require.Error(t, err)
}

func TestRailPDU_Deserialize_HandshakeError(t *testing.T) {
	buf := new(bytes.Buffer)
	// Channel header
	_ = binary.Write(buf, binary.LittleEndian, uint32(8))
	_ = binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))
	// Rail header
	_ = binary.Write(buf, binary.LittleEndian, uint16(RailOrderHandshake))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	// Missing handshake data

	pdu := &RailPDU{}
	err := pdu.Deserialize(buf)
	require.Error(t, err)
}

func TestRailPDU_Deserialize_SysParamError(t *testing.T) {
	buf := new(bytes.Buffer)
	// Channel header
	_ = binary.Write(buf, binary.LittleEndian, uint32(8))
	_ = binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))
	// Rail header
	_ = binary.Write(buf, binary.LittleEndian, uint16(RailOrderSysParam))
	_ = binary.Write(buf, binary.LittleEndian, uint16(17))
	// Missing sysparam data

	pdu := &RailPDU{}
	err := pdu.Deserialize(buf)
	require.Error(t, err)
}

func TestRailPDU_Deserialize_ExecResultError(t *testing.T) {
	buf := new(bytes.Buffer)
	// Channel header
	_ = binary.Write(buf, binary.LittleEndian, uint32(8))
	_ = binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))
	// Rail header
	_ = binary.Write(buf, binary.LittleEndian, uint16(RailOrderExecResult))
	_ = binary.Write(buf, binary.LittleEndian, uint16(32))
	// Missing exec result data

	pdu := &RailPDU{}
	err := pdu.Deserialize(buf)
	require.Error(t, err)
}

func TestRailPDUHeaderDeserialize_ReadError(t *testing.T) {
	header := &RailPDUHeader{}
	err := header.Deserialize(&errorReader{})
	require.Error(t, err)
}

func TestRailPDUHandshakeDeserialize_ReadError(t *testing.T) {
	handshake := &RailPDUHandshake{}
	err := handshake.Deserialize(&errorReader{})
	require.Error(t, err)
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}
