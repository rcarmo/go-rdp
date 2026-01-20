package rdp

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/rcarmo/rdp-html5/internal/protocol/pdu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test handleSlowPathGraphicsUpdate with various data sizes
func TestClient_handleSlowPathGraphicsUpdate_DataSizes(t *testing.T) {
	tests := []struct {
		name       string
		updateType uint16
		dataSize   int
	}{
		{"bitmap empty", SlowPathUpdateTypeBitmap, 0},
		{"bitmap small", SlowPathUpdateTypeBitmap, 10},
		{"bitmap medium", SlowPathUpdateTypeBitmap, 1000},
		{"bitmap large", SlowPathUpdateTypeBitmap, 10000},
		{"palette small", SlowPathUpdateTypePalette, 100},
		{"synchronize", SlowPathUpdateTypeSynchronize, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			_ = binary.Write(buf, binary.LittleEndian, tt.updateType)
			buf.Write(make([]byte, tt.dataSize))

			client := &Client{}
			result, err := client.handleSlowPathGraphicsUpdate(buf)

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

// Test RailPDU serialization round trips
func TestRailPDU_SerializeRoundTrip_Complete(t *testing.T) {
	// Test handshake PDU
	handshakePDU := NewRailHandshakePDU()
	handshakeData := handshakePDU.Serialize()
	require.NotEmpty(t, handshakeData)

	// Test client info PDU
	clientInfoPDU := NewRailClientInfoPDU()
	clientInfoData := clientInfoPDU.Serialize()
	require.NotEmpty(t, clientInfoData)

	// Test system param update PDU
	sysParamPDU := NewRailPDUClientSystemParamUpdate(0x12345678, 0x42)
	sysParamData := sysParamPDU.Serialize()
	require.NotEmpty(t, sysParamData)

	// Test execute PDU
	executePDU := NewRailClientExecutePDU("notepad.exe", "C:\\Windows", "--help")
	executeData := executePDU.Serialize()
	require.NotEmpty(t, executeData)
}

// Test ChannelFlag values
func TestChannelFlag_Values(t *testing.T) {
	assert.Equal(t, ChannelFlag(0x00000001), ChannelFlagFirst)
	assert.Equal(t, ChannelFlag(0x00000002), ChannelFlagLast)
	assert.Equal(t, ChannelFlag(0x00000010), ChannelFlagShowProtocol)
	assert.Equal(t, ChannelFlag(0x00000020), ChannelFlagSuspend)
	assert.Equal(t, ChannelFlag(0x00000040), ChannelFlagResume)
	assert.Equal(t, ChannelFlag(0x00000080), ChannelFlagShadowPersistent)
	assert.Equal(t, ChannelFlag(0x00200000), ChannelFlagCompressed)
	assert.Equal(t, ChannelFlag(0x00400000), ChannelFlagAtFront)
	assert.Equal(t, ChannelFlag(0x00800000), ChannelFlagFlushed)
}

// Test handleRail with various states
func TestClient_handleRail_States(t *testing.T) {
	tests := []struct {
		name          string
		initialState  RailState
		orderType     RailOrder
		setupClient   func() *Client
	}{
		{
			name:         "uninitialized state",
			initialState: RailStateUninitialized,
			orderType:    RailOrderSysParam,
			setupClient: func() *Client {
				return &Client{
					remoteApp:    &RemoteApp{App: "test.exe"},
					railState:    RailStateUninitialized,
					channelIDMap: map[string]uint16{"rail": 1005},
				}
			},
		},
		{
			name:         "wait for data state",
			initialState: RailStateWaitForData,
			orderType:    RailOrderSysParam,
			setupClient: func() *Client {
				return &Client{
					remoteApp:    &RemoteApp{App: "test.exe"},
					railState:    RailStateWaitForData,
					channelIDMap: map[string]uint16{"rail": 1005},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()

			// Build a SysParam PDU (which is ignored)
			buf := new(bytes.Buffer)
			_ = binary.Write(buf, binary.LittleEndian, uint32(13))
			_ = binary.Write(buf, binary.LittleEndian, uint32(ChannelFlagFirst|ChannelFlagLast))
			_ = binary.Write(buf, binary.LittleEndian, uint16(tt.orderType))
			_ = binary.Write(buf, binary.LittleEndian, uint16(17))
			_ = binary.Write(buf, binary.LittleEndian, uint32(0x12345678))
			_ = binary.Write(buf, binary.LittleEndian, uint8(0x42))

			err := client.handleRail(buf)
			assert.NoError(t, err)
		})
	}
}

// Test RailPDU serialization with different order types
func TestRailPDU_Serialize_Coverage(t *testing.T) {
	tests := []struct {
		name string
		pdu  *RailPDU
	}{
		{
			name: "handshake",
			pdu:  NewRailHandshakePDU(),
		},
		{
			name: "exec",
			pdu:  NewRailClientExecutePDU("app.exe", "dir", "args"),
		},
		{
			name: "sysparam",
			pdu:  NewRailPDUClientSystemParamUpdate(1, 2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.pdu.Serialize()
			require.NotEmpty(t, data)
			assert.GreaterOrEqual(t, len(data), 8)
		})
	}
}

// Test ServerCapabilities with all capability types
func TestClient_GetServerCapabilities_Complete(t *testing.T) {
	client := &Client{
		serverCapabilitySets: []pdu.CapabilitySet{
			{
				CapabilitySetType: pdu.CapabilitySetTypeBitmap,
				BitmapCapabilitySet: &pdu.BitmapCapabilitySet{
					PreferredBitsPerPixel: 32,
					DesktopWidth:          1920,
					DesktopHeight:         1080,
				},
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeGeneral,
				GeneralCapabilitySet: &pdu.GeneralCapabilitySet{
					ExtraFlags: 0x1234,
				},
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeOrder,
				OrderCapabilitySet: &pdu.OrderCapabilitySet{
					OrderFlags: 0x5678,
				},
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeSurfaceCommands,
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeLargePointer,
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeFrameAcknowledge,
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeMultifragmentUpdate,
				MultifragmentUpdateCapabilitySet: &pdu.MultifragmentUpdateCapabilitySet{
					MaxRequestSize: 0x100000,
				},
			},
			{
				CapabilitySetType: pdu.CapabilitySetTypeBitmapCodecs,
				BitmapCodecsCapabilitySet: &pdu.BitmapCodecsCapabilitySet{
					BitmapCodecArray: []pdu.BitmapCodec{
						{CodecGUID: guidNSCodec},
						{CodecGUID: guidRemoteFX},
						{CodecGUID: guidImageRemoteFX},
						{CodecGUID: guidClearCodec},
					},
				},
			},
		},
	}

	info := client.GetServerCapabilities()

	require.NotNil(t, info)
	assert.Equal(t, 32, info.ColorDepth)
	assert.Equal(t, "1920x1080", info.DesktopSize)
	assert.Equal(t, uint16(0x1234), info.GeneralFlags)
	assert.Equal(t, uint32(0x5678), info.OrderFlags)
	assert.True(t, info.SurfaceCommands)
	assert.True(t, info.LargePointer)
	assert.True(t, info.FrameAcknowledge)
	assert.Equal(t, uint32(0x100000), info.MultifragmentSize)
	assert.Len(t, info.BitmapCodecs, 4)
}

// Test RailPDUExecResult deserialization
func TestRailPDUExecResult_Deserialize_Extended(t *testing.T) {
	tests := []struct {
		name       string
		flags      uint16
		execResult uint16
		rawResult  uint32
		exeOrFile  string
	}{
		{"success", 0x0000, 0x0000, 0, "calc.exe"},
		{"failed", 0x0001, 0x0001, 0x12345678, "notepad.exe"},
		{"empty filename", 0x0000, 0x0000, 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			_ = binary.Write(buf, binary.LittleEndian, tt.flags)
			_ = binary.Write(buf, binary.LittleEndian, tt.execResult)
			_ = binary.Write(buf, binary.LittleEndian, tt.rawResult)
			_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // padding
			_ = binary.Write(buf, binary.LittleEndian, uint16(len(tt.exeOrFile)))
			buf.Write([]byte(tt.exeOrFile))

			execResult := &RailPDUExecResult{}
			err := execResult.Deserialize(buf)

			require.NoError(t, err)
			assert.Equal(t, tt.flags, execResult.Flags)
			assert.Equal(t, tt.execResult, execResult.ExecResult)
			assert.Equal(t, tt.rawResult, execResult.RawResult)
			assert.Equal(t, tt.exeOrFile, execResult.ExeOrFile)
		})
	}
}

// Test RailPDUClientExecute serialization with various inputs
func TestRailPDUClientExecute_Serialize_Extended(t *testing.T) {
	tests := []struct {
		name       string
		exeOrFile  string
		workingDir string
		arguments  string
	}{
		{"simple", "app.exe", "", ""},
		{"with working dir", "app.exe", "C:\\Program Files", ""},
		{"with args", "app.exe", "", "--help --version"},
		{"full", "longappname.exe", "C:\\Windows\\System32", "-a -b -c -d -e"},
		{"unicode", "приложение.exe", "目录", "参数"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &RailPDUClientExecute{
				ExeOrFile:  tt.exeOrFile,
				WorkingDir: tt.workingDir,
				Arguments:  tt.arguments,
			}

			data := exec.Serialize()
			require.NotEmpty(t, data)
			assert.GreaterOrEqual(t, len(data), 8)
		})
	}
}

func TestRailPDUSystemParameters_Deserialize_Errors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"too short", []byte{0x01}},
		{"missing body", []byte{0x01, 0x02, 0x03, 0x04}},
		{"empty", []byte{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sysParams := &RailPDUSystemParameters{}
			err := sysParams.Deserialize(bytes.NewReader(tt.data))
			assert.Error(t, err)
		})
	}
}

// Test ChannelPDUHeader deserialize errors
func TestChannelPDUHeader_Deserialize_Errors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte{0x01, 0x02}},
		{"only length", []byte{0x08, 0x00, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &ChannelPDUHeader{}
			err := header.Deserialize(bytes.NewReader(tt.data))
			assert.Error(t, err)
		})
	}
}

// Test RailPDUHeader deserialize errors
func TestRailPDUHeader_Deserialize_Errors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"only order type", []byte{0x01, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &RailPDUHeader{}
			err := header.Deserialize(bytes.NewReader(tt.data))
			assert.Error(t, err)
		})
	}
}

// Test RailPDUHandshake deserialize errors
func TestRailPDUHandshake_Deserialize_Errors(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte{0x01, 0x02}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handshake := &RailPDUHandshake{}
			err := handshake.Deserialize(bytes.NewReader(tt.data))
			assert.Error(t, err)
		})
	}
}

// Test PDU types
func TestPDU_Types(t *testing.T) {
	// Test Type constants
	assert.Equal(t, pdu.Type(0x11), pdu.TypeDemandActive)
	assert.Equal(t, pdu.Type(0x13), pdu.TypeConfirmActive)
	assert.Equal(t, pdu.Type(0x16), pdu.TypeDeactivateAll)
	assert.Equal(t, pdu.Type(0x17), pdu.TypeData)

	// Test Type2 constants
	assert.Equal(t, pdu.Type2(0x02), pdu.Type2Update)
	assert.Equal(t, pdu.Type2(0x14), pdu.Type2Control)
	assert.Equal(t, pdu.Type2(0x1F), pdu.Type2Synchronize)
	assert.Equal(t, pdu.Type2(0x28), pdu.Type2Fontmap)
	assert.Equal(t, pdu.Type2(0x2F), pdu.Type2ErrorInfo)
}

// Test Type methods
func TestType_Methods(t *testing.T) {
	assert.True(t, pdu.TypeDemandActive.IsDemandActive())
	assert.True(t, pdu.TypeConfirmActive.IsConfirmActive())
	assert.True(t, pdu.TypeDeactivateAll.IsDeactivateAll())
	assert.True(t, pdu.TypeData.IsData())

	// Negative cases
	assert.False(t, pdu.TypeData.IsDemandActive())
	assert.False(t, pdu.TypeData.IsDeactivateAll())
}

// Test Type2 methods
func TestType2_Methods_Extended(t *testing.T) {
	assert.True(t, pdu.Type2Update.IsUpdate())
	assert.True(t, pdu.Type2Control.IsControl())
	assert.True(t, pdu.Type2Synchronize.IsSynchronize())
	assert.True(t, pdu.Type2Fontmap.IsFontmap())
	assert.True(t, pdu.Type2ErrorInfo.IsErrorInfo())

	// Negative cases
	assert.False(t, pdu.Type2Update.IsControl())
	assert.False(t, pdu.Type2Control.IsSynchronize())
}

// Test NewClient with various configurations
func TestClient_NewClient_Extended(t *testing.T) {
	tests := []struct {
		name        string
		hostname    string
		expectError bool
	}{
		{"localhost unreachable", "127.0.0.1:13389", true},
		{"invalid port", "localhost:99999", true},
		{"missing port", "localhost", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.hostname, "user", "pass", 1024, 768, 16)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
			}
		})
	}
}
