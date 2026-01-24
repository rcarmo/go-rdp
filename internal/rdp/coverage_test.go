package rdp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/rcarmo/go-rdp/internal/protocol/pdu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMCSLayer implements a mock for MCS layer testing
type MockMCSLayer struct {
	SendFunc        func(userID, channelID uint16, data []byte) error
	ReceiveFunc     func() (channelID uint16, reader io.Reader, err error)
	ConnectFunc     func(userData []byte) (io.Reader, error)
	ErectDomainFunc func() error
	AttachUserFunc  func() (uint16, error)
	JoinChannelsFunc func(userID uint16, channelIDMap map[string]uint16) error

	SendCalls         []mockSendCall
	ReceiveCalls      int
	ConnectCalls      [][]byte
	ErectDomainCalls  int
	AttachUserCalls   int
	JoinChannelsCalls []mockJoinChannelsCall
}

type mockSendCall struct {
	UserID    uint16
	ChannelID uint16
	Data      []byte
}

type mockJoinChannelsCall struct {
	UserID       uint16
	ChannelIDMap map[string]uint16
}

func (m *MockMCSLayer) Send(userID, channelID uint16, data []byte) error {
	m.SendCalls = append(m.SendCalls, mockSendCall{userID, channelID, data})
	if m.SendFunc != nil {
		return m.SendFunc(userID, channelID, data)
	}
	return nil
}

func (m *MockMCSLayer) Receive() (uint16, io.Reader, error) {
	m.ReceiveCalls++
	if m.ReceiveFunc != nil {
		return m.ReceiveFunc()
	}
	return 0, nil, nil
}

func (m *MockMCSLayer) Connect(userData []byte) (io.Reader, error) {
	m.ConnectCalls = append(m.ConnectCalls, userData)
	if m.ConnectFunc != nil {
		return m.ConnectFunc(userData)
	}
	return nil, nil
}

func (m *MockMCSLayer) ErectDomain() error {
	m.ErectDomainCalls++
	if m.ErectDomainFunc != nil {
		return m.ErectDomainFunc()
	}
	return nil
}

func (m *MockMCSLayer) AttachUser() (uint16, error) {
	m.AttachUserCalls++
	if m.AttachUserFunc != nil {
		return m.AttachUserFunc()
	}
	return 1001, nil
}

func (m *MockMCSLayer) JoinChannels(userID uint16, channelIDMap map[string]uint16) error {
	m.JoinChannelsCalls = append(m.JoinChannelsCalls, mockJoinChannelsCall{userID, channelIDMap})
	if m.JoinChannelsFunc != nil {
		return m.JoinChannelsFunc(userID, channelIDMap)
	}
	return nil
}

// TestReceiveProtocol_EdgeCases tests edge cases for receiveProtocol
func TestReceiveProtocol_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantX224  bool
		wantFP    bool
	}{
		{
			name:     "Protocol code 1 is not fastpath (bit 0 set)",
			data:     []byte{0x01, 0x02},
			wantX224: false,
			wantFP:   false,
		},
		{
			name:     "Protocol code 2 is not fastpath (bit 1 set)",
			data:     []byte{0x02, 0x02},
			wantX224: false,
			wantFP:   false,
		},
		{
			name:     "Protocol code 0x84 is fastpath",
			data:     []byte{0x84, 0x02},
			wantX224: false,
			wantFP:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(bytes.NewReader(tt.data))
			result, err := receiveProtocol(reader)

			require.NoError(t, err)
			assert.Equal(t, tt.wantX224, result.IsX224())
			assert.Equal(t, tt.wantFP, result.IsFastpath())
		})
	}
}

// Test CapabilitySetType coverage
func TestCapabilitySetTypes(t *testing.T) {
	types := []pdu.CapabilitySetType{
		pdu.CapabilitySetTypeGeneral,
		pdu.CapabilitySetTypeBitmap,
		pdu.CapabilitySetTypeOrder,
		pdu.CapabilitySetTypeBitmapCache,
		pdu.CapabilitySetTypeControl,
		pdu.CapabilitySetTypePointer,
		pdu.CapabilitySetTypeInput,
		pdu.CapabilitySetTypeBrush,
		pdu.CapabilitySetTypeGlyphCache,
		pdu.CapabilitySetTypeOffscreenBitmapCache,
		pdu.CapabilitySetTypeVirtualChannel,
		pdu.CapabilitySetTypeSound,
		pdu.CapabilitySetTypeSurfaceCommands,
		pdu.CapabilitySetTypeBitmapCodecs,
		pdu.CapabilitySetTypeMultifragmentUpdate,
		pdu.CapabilitySetTypeLargePointer,
		pdu.CapabilitySetTypeFrameAcknowledge,
	}

	for _, capType := range types {
		assert.NotZero(t, capType, "Capability set type should not be zero")
	}
}

// TestServerDemandActive_Deserialize tests ServerDemandActive deserialization
func TestServerDemandActive_Deserialize_Basic(t *testing.T) {
	buf := new(bytes.Buffer)

	// ShareControlHeader
	_ = binary.Write(buf, binary.LittleEndian, uint16(26)) // totalLength
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x11)) // pduType
	_ = binary.Write(buf, binary.LittleEndian, uint16(1001)) // pduSource

	// DemandActivePDU data
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x00010002)) // shareId
	_ = binary.Write(buf, binary.LittleEndian, uint16(4))          // lengthSourceDescriptor
	buf.Write([]byte("RDP\x00"))                               // sourceDescriptor
	_ = binary.Write(buf, binary.LittleEndian, uint16(4))          // lengthCombinedCapabilities
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))          // numberCapabilities
	_ = binary.Write(buf, binary.LittleEndian, uint16(0))          // pad2Octets
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))          // sessionId

	var resp pdu.ServerDemandActive
	err := resp.Deserialize(buf)

	require.NoError(t, err)
	assert.Equal(t, uint32(0x00010002), resp.ShareID)
}

// TestNewClientConfirmActive tests ClientConfirmActive PDU creation
func TestNewClientConfirmActive_Creation(t *testing.T) {
	tests := []struct {
		name      string
		shareID   uint32
		userID    uint16
		width     uint16
		height    uint16
		remoteApp bool
	}{
		{
			name:      "standard desktop",
			shareID:   0x12345678,
			userID:    1001,
			width:     1920,
			height:    1080,
			remoteApp: false,
		},
		{
			name:      "remote app",
			shareID:   0xABCDEF01,
			userID:    1005,
			width:     800,
			height:    600,
			remoteApp: true,
		},
		{
			name:      "small resolution",
			shareID:   1,
			userID:    1001,
			width:     640,
			height:    480,
			remoteApp: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pdu := pdu.NewClientConfirmActive(tt.shareID, tt.userID, tt.width, tt.height, tt.remoteApp)
			serialized := pdu.Serialize()

			require.NotEmpty(t, serialized)
			assert.Greater(t, len(serialized), 6)
		})
	}
}

// TestNewSynchronize_Creation tests Synchronize PDU creation
func TestNewSynchronize_Creation(t *testing.T) {
	tests := []struct {
		shareID uint32
		userID  uint16
	}{
		{0x12345678, 1001},
		{1, 1},
		{0xFFFFFFFF, 65535},
	}

	for _, tt := range tests {
		pdu := pdu.NewSynchronize(tt.shareID, tt.userID)
		serialized := pdu.Serialize()
		require.NotEmpty(t, serialized)
	}
}

// TestNewControl_Creation tests Control PDU creation
func TestNewControl_Creation(t *testing.T) {
	tests := []struct {
		name    string
		action  pdu.ControlAction
	}{
		{"cooperate", pdu.ControlActionCooperate},
		{"request control", pdu.ControlActionRequestControl},
		{"granted control", pdu.ControlActionGrantedControl},
		{"detach", pdu.ControlActionDetach},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlPDU := pdu.NewControl(0x12345678, 1001, tt.action)
			serialized := controlPDU.Serialize()
			require.NotEmpty(t, serialized)
		})
	}
}

// TestNewFontList_Creation tests FontList PDU creation
func TestNewFontList_Creation(t *testing.T) {
	fontList := pdu.NewFontList(0x12345678, 1001)
	serialized := fontList.Serialize()
	require.NotEmpty(t, serialized)
}

// TestType2_IsMethods tests PDU Type2 detection methods
func TestType2_IsMethods(t *testing.T) {
	tests := []struct {
		name      string
		pduType2  pdu.Type2
		check     func(pdu.Type2) bool
		expected  bool
	}{
		{"synchronize check", pdu.Type2(0x1F), func(t pdu.Type2) bool { return t.IsSynchronize() }, true},
		{"control check", pdu.Type2(0x14), func(t pdu.Type2) bool { return t.IsControl() }, true},
		{"fontmap check", pdu.Type2(0x28), func(t pdu.Type2) bool { return t.IsFontmap() }, true},
		{"update check", pdu.Type2(0x02), func(t pdu.Type2) bool { return t.IsUpdate() }, true},
		{"error info check", pdu.Type2(0x2F), func(t pdu.Type2) bool { return t.IsErrorInfo() }, true},
		{"not synchronize", pdu.Type2(0x14), func(t pdu.Type2) bool { return t.IsSynchronize() }, false},
		{"not control", pdu.Type2(0x1F), func(t pdu.Type2) bool { return t.IsControl() }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check(tt.pduType2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestType_IsDeactivateAll tests ShareControlHeader PDU type
func TestType_IsDeactivateAll(t *testing.T) {
	tests := []struct {
		name     string
		pduType  pdu.Type
		expected bool
	}{
		{"deactivate all", pdu.TypeDeactivateAll, true},
		{"data PDU", pdu.TypeData, false},
		{"demand active", pdu.TypeDemandActive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.pduType.IsDeactivateAll())
		})
	}
}

// TestNewClientInfo_Creation tests ClientInfo PDU creation
func TestNewClientInfo_Creation(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		username string
		password string
	}{
		{"basic", "", "user", "pass"},
		{"with domain", "DOMAIN", "user", "pass123"},
		{"empty password", "", "user", ""},
		{"unicode", "", "用户", "密码"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientInfo := pdu.NewClientInfo(tt.domain, tt.username, tt.password)

			enhanced := clientInfo.Serialize(true)
			basic := clientInfo.Serialize(false)

			require.NotEmpty(t, enhanced)
			require.NotEmpty(t, basic)
			assert.GreaterOrEqual(t, len(basic), len(enhanced))
		})
	}
}

// TestNegotiationProtocol_Methods tests negotiation protocol methods
func TestNegotiationProtocol_Methods(t *testing.T) {
	tests := []struct {
		name     string
		protocol pdu.NegotiationProtocol
		isSSL    bool
		isHybrid bool
		isRDP    bool
	}{
		{"SSL", pdu.NegotiationProtocolSSL, true, false, false},
		{"Hybrid", pdu.NegotiationProtocolHybrid, false, true, false},
		{"RDP", pdu.NegotiationProtocolRDP, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isSSL, tt.protocol.IsSSL())
			assert.Equal(t, tt.isHybrid, tt.protocol.IsHybrid())
			assert.Equal(t, tt.isRDP, tt.protocol.IsRDP())
		})
	}
}

// TestServerLicenseError_Deserialize tests licensing PDU deserialization
func TestServerLicenseError_Deserialize(t *testing.T) {
	tests := []struct {
		name    string
		msgType uint8
	}{
		{"ERROR_ALERT", 0xFF},
		{"NEW_LICENSE", 0x03},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)

			// Security header with SEC_LICENSE_PKT flag (0x0080)
			_ = binary.Write(buf, binary.LittleEndian, uint32(0x0080))

			// Preamble
			_ = binary.Write(buf, binary.LittleEndian, tt.msgType)
			_ = binary.Write(buf, binary.LittleEndian, uint8(0x02))
			_ = binary.Write(buf, binary.LittleEndian, uint16(20))

			// ValidClientMessage
			_ = binary.Write(buf, binary.LittleEndian, uint32(0x00000007))
			_ = binary.Write(buf, binary.LittleEndian, uint32(0x00000002))
			_ = binary.Write(buf, binary.LittleEndian, uint16(0))
			_ = binary.Write(buf, binary.LittleEndian, uint16(0))

			var resp pdu.ServerLicenseError
			err := resp.Deserialize(buf, true)

			require.NoError(t, err)
			assert.Equal(t, tt.msgType, resp.Preamble.MsgType)
		})
	}
}

// TestData_Deserialize tests Data PDU deserialization
func TestData_Deserialize_Extended(t *testing.T) {
	tests := []struct {
		name     string
		pduType2 uint8
	}{
		{"synchronize", 0x1F},
		{"control", 0x14},
		{"fontmap", 0x28},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)

			// ShareControlHeader
			_ = binary.Write(buf, binary.LittleEndian, uint16(30))
			_ = binary.Write(buf, binary.LittleEndian, uint16(0x17))
			_ = binary.Write(buf, binary.LittleEndian, uint16(1001))

			// ShareDataHeader
			_ = binary.Write(buf, binary.LittleEndian, uint32(0x12345678))
			_ = binary.Write(buf, binary.LittleEndian, uint8(0))
			_ = binary.Write(buf, binary.LittleEndian, uint8(1))
			_ = binary.Write(buf, binary.LittleEndian, uint16(14))
			_ = binary.Write(buf, binary.LittleEndian, tt.pduType2)
			_ = binary.Write(buf, binary.LittleEndian, uint8(0))
			_ = binary.Write(buf, binary.LittleEndian, uint16(0))

			// PDU-specific data
			switch tt.pduType2 {
			case 0x1F: // Synchronize
				_ = binary.Write(buf, binary.LittleEndian, uint16(1))
				_ = binary.Write(buf, binary.LittleEndian, uint16(1001))
			case 0x14: // Control
				_ = binary.Write(buf, binary.LittleEndian, uint16(4)) // action=COOPERATE
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))
				_ = binary.Write(buf, binary.LittleEndian, uint32(0))
			case 0x28: // Fontmap
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))
			}

			dataPDU := &pdu.Data{}
			err := dataPDU.Deserialize(buf)
			require.NoError(t, err)
		})
	}
}

// TestErrorDefinitions tests error definitions exist
func TestErrorDefinitions(t *testing.T) {
	assert.NotNil(t, ErrUnsupportedRequestedProtocol)
	assert.NotNil(t, pdu.ErrDeactivateAll)
}

// TestConnectionConstants tests connection-related constants
func TestConnectionConstants_Extended(t *testing.T) {
	assert.Equal(t, 64*1024, readBufferSize)
	assert.NotZero(t, tcpConnectionTimeout)
}

// TestClientUserDataSet tests client user data set creation
func TestClientUserDataSet_Creation(t *testing.T) {
	tests := []struct {
		name     string
		protocol uint32
		width    uint16
		height   uint16
		depth    int
		channels []string
	}{
		{"basic", 1, 1024, 768, 16, nil},
		{"with channels", 1, 1920, 1080, 24, []string{"cliprdr", "rdpdr"}},
		{"high depth", 2, 2560, 1440, 32, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userData := pdu.NewClientUserDataSet(tt.protocol, tt.width, tt.height, tt.depth, tt.channels)
			serialized := userData.Serialize()
			require.NotEmpty(t, serialized)
		})
	}
}

// TestShareControlHeader_Deserialize tests ShareControlHeader deserialization
func TestShareControlHeader_Deserialize(t *testing.T) {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, uint16(20))   // totalLength
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x17)) // pduType
	_ = binary.Write(buf, binary.LittleEndian, uint16(1001)) // pduSource

	var header pdu.ShareControlHeader
	err := header.Deserialize(buf)
	require.NoError(t, err)
}

// TestNegotiationFailureCode_String tests failure code string representations
func TestNegotiationFailureCode_String(t *testing.T) {
	codes := []pdu.NegotiationFailureCode{
		pdu.NegotiationFailureCodeHybridRequired,
		pdu.NegotiationFailureCodeSSLRequired,
		pdu.NegotiationFailureCodeSSLWithUserAuthRequired,
	}

	for _, code := range codes {
		str := code.String()
		assert.NotEmpty(t, str)
	}
}

// TestNegotiationType_IsFailure tests negotiation type failure check
func TestNegotiationType_IsFailure(t *testing.T) {
	tests := []struct {
		name      string
		negType   pdu.NegotiationType
		isFailure bool
	}{
		{"failure", pdu.NegotiationTypeFailure, true},
		{"response", pdu.NegotiationTypeResponse, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isFailure, tt.negType.IsFailure())
		})
	}
}
