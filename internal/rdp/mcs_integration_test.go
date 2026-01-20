package rdp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/rcarmo/rdp-html5/internal/protocol/pdu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testMCSLayer is a mock MCS layer for testing
type testMCSLayer struct {
	sendFunc        func(userID, channelID uint16, data []byte) error
	receiveFunc     func() (uint16, io.Reader, error)
	connectFunc     func(userData []byte) (io.Reader, error)
	erectDomainFunc func() error
	attachUserFunc  func() (uint16, error)
	joinChannelsFunc func(userID uint16, channelIDMap map[string]uint16) error
	
	sendCalls []sendCall
}

type sendCall struct {
	userID    uint16
	channelID uint16
	data      []byte
}

func (m *testMCSLayer) Send(userID, channelID uint16, data []byte) error {
	m.sendCalls = append(m.sendCalls, sendCall{userID, channelID, data})
	if m.sendFunc != nil {
		return m.sendFunc(userID, channelID, data)
	}
	return nil
}

func (m *testMCSLayer) Receive() (uint16, io.Reader, error) {
	if m.receiveFunc != nil {
		return m.receiveFunc()
	}
	return 0, nil, nil
}

func (m *testMCSLayer) Connect(userData []byte) (io.Reader, error) {
	if m.connectFunc != nil {
		return m.connectFunc(userData)
	}
	return bytes.NewReader([]byte{}), nil
}

func (m *testMCSLayer) ErectDomain() error {
	if m.erectDomainFunc != nil {
		return m.erectDomainFunc()
	}
	return nil
}

func (m *testMCSLayer) AttachUser() (uint16, error) {
	if m.attachUserFunc != nil {
		return m.attachUserFunc()
	}
	return 1001, nil
}

func (m *testMCSLayer) JoinChannels(userID uint16, channelIDMap map[string]uint16) error {
	if m.joinChannelsFunc != nil {
		return m.joinChannelsFunc(userID, channelIDMap)
	}
	return nil
}

// Test capabilitiesExchange with mock MCS layer
func TestClient_capabilitiesExchange_Success(t *testing.T) {
	// Build a ServerDemandActive PDU response
	demandActiveBuf := new(bytes.Buffer)
	
	// ShareControlHeader
	_ = binary.Write(demandActiveBuf, binary.LittleEndian, uint16(40)) // totalLength
	_ = binary.Write(demandActiveBuf, binary.LittleEndian, uint16(0x11)) // pduType (demand active)
	_ = binary.Write(demandActiveBuf, binary.LittleEndian, uint16(1001)) // pduSource
	
	// DemandActivePDU
	_ = binary.Write(demandActiveBuf, binary.LittleEndian, uint32(0x12345678)) // shareId
	_ = binary.Write(demandActiveBuf, binary.LittleEndian, uint16(4)) // lengthSourceDescriptor
	demandActiveBuf.Write([]byte("RDP\x00")) // sourceDescriptor
	_ = binary.Write(demandActiveBuf, binary.LittleEndian, uint16(4)) // lengthCombinedCapabilities
	_ = binary.Write(demandActiveBuf, binary.LittleEndian, uint16(0)) // numberCapabilities
	_ = binary.Write(demandActiveBuf, binary.LittleEndian, uint16(0)) // pad2Octets
	_ = binary.Write(demandActiveBuf, binary.LittleEndian, uint32(0)) // sessionId

	mockMCS := &testMCSLayer{
		receiveFunc: func() (uint16, io.Reader, error) {
			return 1003, bytes.NewReader(demandActiveBuf.Bytes()), nil
		},
	}

	client := &Client{
		mcsLayer:     mockMCS,
		userID:       1001,
		channelIDMap: map[string]uint16{"global": 1003},
		desktopWidth:  1024,
		desktopHeight: 768,
	}

	err := client.capabilitiesExchange()
	require.NoError(t, err)
	assert.Equal(t, uint32(0x12345678), client.shareID)
	assert.Len(t, mockMCS.sendCalls, 1)
}

// Test capabilitiesExchange with receive error
func TestClient_capabilitiesExchange_ReceiveError(t *testing.T) {
	mockMCS := &testMCSLayer{
		receiveFunc: func() (uint16, io.Reader, error) {
			return 0, nil, errors.New("receive error")
		},
	}

	client := &Client{
		mcsLayer: mockMCS,
	}

	err := client.capabilitiesExchange()
	assert.Error(t, err)
}

// Test channelConnection success
func TestClient_channelConnection_Success(t *testing.T) {
	mockMCS := &testMCSLayer{}

	client := &Client{
		mcsLayer:        mockMCS,
		channelIDMap:    map[string]uint16{"global": 1003},
		skipChannelJoin: false,
	}

	err := client.channelConnection()
	require.NoError(t, err)
	assert.Equal(t, uint16(1001), client.userID)
	assert.Equal(t, uint16(1001), client.channelIDMap["user"])
}

// Test channelConnection with skipChannelJoin
func TestClient_channelConnection_SkipJoin(t *testing.T) {
	mockMCS := &testMCSLayer{}

	client := &Client{
		mcsLayer:        mockMCS,
		channelIDMap:    map[string]uint16{"global": 1003},
		skipChannelJoin: true,
	}

	err := client.channelConnection()
	require.NoError(t, err)
	assert.Equal(t, uint16(1001), client.userID)
}

// Test channelConnection with ErectDomain error
func TestClient_channelConnection_ErectDomainError(t *testing.T) {
	mockMCS := &testMCSLayer{
		erectDomainFunc: func() error {
			return errors.New("erect domain error")
		},
	}

	client := &Client{
		mcsLayer:     mockMCS,
		channelIDMap: map[string]uint16{"global": 1003},
	}

	err := client.channelConnection()
	assert.Error(t, err)
}

// Test channelConnection with AttachUser error
func TestClient_channelConnection_AttachUserError(t *testing.T) {
	mockMCS := &testMCSLayer{
		attachUserFunc: func() (uint16, error) {
			return 0, errors.New("attach user error")
		},
	}

	client := &Client{
		mcsLayer:     mockMCS,
		channelIDMap: map[string]uint16{"global": 1003},
	}

	err := client.channelConnection()
	assert.Error(t, err)
}

// Test channelConnection with JoinChannels error
func TestClient_channelConnection_JoinChannelsError(t *testing.T) {
	mockMCS := &testMCSLayer{
		joinChannelsFunc: func(userID uint16, channelIDMap map[string]uint16) error {
			return errors.New("join channels error")
		},
	}

	client := &Client{
		mcsLayer:        mockMCS,
		channelIDMap:    map[string]uint16{"global": 1003},
		skipChannelJoin: false,
	}

	err := client.channelConnection()
	assert.Error(t, err)
}

// Test secureSettingsExchange success
func TestClient_secureSettingsExchange_Success(t *testing.T) {
	mockMCS := &testMCSLayer{}

	client := &Client{
		mcsLayer:         mockMCS,
		userID:           1001,
		channelIDMap:     map[string]uint16{"global": 1003},
		domain:           "DOMAIN",
		username:         "user",
		password:         "pass",
		selectedProtocol: pdu.NegotiationProtocolSSL,
	}

	err := client.secureSettingsExchange()
	require.NoError(t, err)
	assert.Len(t, mockMCS.sendCalls, 1)
}

// Test secureSettingsExchange with remoteApp
func TestClient_secureSettingsExchange_WithRemoteApp(t *testing.T) {
	mockMCS := &testMCSLayer{}

	client := &Client{
		mcsLayer:         mockMCS,
		userID:           1001,
		channelIDMap:     map[string]uint16{"global": 1003},
		domain:           "",
		username:         "user",
		password:         "pass",
		selectedProtocol: pdu.NegotiationProtocolHybrid,
		remoteApp:        &RemoteApp{App: "notepad.exe"},
	}

	err := client.secureSettingsExchange()
	require.NoError(t, err)
}

// Test secureSettingsExchange with send error
func TestClient_secureSettingsExchange_SendError(t *testing.T) {
	mockMCS := &testMCSLayer{
		sendFunc: func(userID, channelID uint16, data []byte) error {
			return errors.New("send error")
		},
	}

	client := &Client{
		mcsLayer:     mockMCS,
		userID:       1001,
		channelIDMap: map[string]uint16{"global": 1003},
		username:     "user",
		password:     "pass",
	}

	err := client.secureSettingsExchange()
	assert.Error(t, err)
}

// Test connectionFinalization with synchronize PDU
func TestClient_connectionFinalization_SynchronizeReceived(t *testing.T) {
	callCount := 0
	
	mockMCS := &testMCSLayer{
		receiveFunc: func() (uint16, io.Reader, error) {
			callCount++
			buf := new(bytes.Buffer)
			
			// ShareControlHeader
			_ = binary.Write(buf, binary.LittleEndian, uint16(22))
			_ = binary.Write(buf, binary.LittleEndian, uint16(0x17)) // PDUTYPE_DATAPDU
			_ = binary.Write(buf, binary.LittleEndian, uint16(1001))
			
			// ShareDataHeader
			_ = binary.Write(buf, binary.LittleEndian, uint32(0x12345678))
			_ = binary.Write(buf, binary.LittleEndian, uint8(0))
			_ = binary.Write(buf, binary.LittleEndian, uint8(1))
			_ = binary.Write(buf, binary.LittleEndian, uint16(8))
			
			var pduType2 uint8
			switch callCount {
			case 1:
				pduType2 = 0x1F // PDUTYPE2_SYNCHRONIZE
			case 2:
				pduType2 = 0x14 // PDUTYPE2_CONTROL (cooperate)
			case 3:
				pduType2 = 0x14 // PDUTYPE2_CONTROL (granted)
			case 4:
				pduType2 = 0x28 // PDUTYPE2_FONTMAP
			default:
				return 0, nil, io.EOF
			}
			
			_ = binary.Write(buf, binary.LittleEndian, pduType2)
			_ = binary.Write(buf, binary.LittleEndian, uint8(0))
			_ = binary.Write(buf, binary.LittleEndian, uint16(0))
			
			// PDU data
			switch pduType2 {
			case 0x1F: // Synchronize
				_ = binary.Write(buf, binary.LittleEndian, uint16(1))
				_ = binary.Write(buf, binary.LittleEndian, uint16(1001))
			case 0x14: // Control
				if callCount == 2 {
					_ = binary.Write(buf, binary.LittleEndian, uint16(4)) // cooperate
				} else {
					_ = binary.Write(buf, binary.LittleEndian, uint16(2)) // granted
				}
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))
				_ = binary.Write(buf, binary.LittleEndian, uint32(0))
			case 0x28: // Fontmap
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))
				_ = binary.Write(buf, binary.LittleEndian, uint16(0))
			}
			
			return 1003, buf, nil
		},
	}

	client := &Client{
		mcsLayer:     mockMCS,
		shareID:      0x12345678,
		userID:       1001,
		channelIDMap: map[string]uint16{"global": 1003},
	}

	err := client.connectionFinalization()
	require.NoError(t, err)
	assert.Equal(t, 4, callCount)
}

// Test connectionFinalization with receive error
func TestClient_connectionFinalization_ReceiveError(t *testing.T) {
	mockMCS := &testMCSLayer{
		receiveFunc: func() (uint16, io.Reader, error) {
			return 0, nil, errors.New("receive error")
		},
	}

	client := &Client{
		mcsLayer:     mockMCS,
		shareID:      0x12345678,
		userID:       1001,
		channelIDMap: map[string]uint16{"global": 1003},
	}

	err := client.connectionFinalization()
	assert.Error(t, err)
}

// Test sendRefreshRect success
func TestClient_sendRefreshRect_Success(t *testing.T) {
	mockMCS := &testMCSLayer{}

	client := &Client{
		mcsLayer:      mockMCS,
		shareID:       0x12345678,
		userID:        1001,
		channelIDMap:  map[string]uint16{"global": 1003},
		desktopWidth:  1024,
		desktopHeight: 768,
	}

	err := client.sendRefreshRect()
	require.NoError(t, err)
	assert.Len(t, mockMCS.sendCalls, 1)
}

// Test sendRefreshRect with send error
func TestClient_sendRefreshRect_SendError(t *testing.T) {
	mockMCS := &testMCSLayer{
		sendFunc: func(userID, channelID uint16, data []byte) error {
			return errors.New("send error")
		},
	}

	client := &Client{
		mcsLayer:      mockMCS,
		shareID:       0x12345678,
		userID:        1001,
		channelIDMap:  map[string]uint16{"global": 1003},
		desktopWidth:  1024,
		desktopHeight: 768,
	}

	err := client.sendRefreshRect()
	assert.Error(t, err)
}

// Test licensing with valid license error
func TestClient_licensing_ValidClient(t *testing.T) {
	mockMCS := &testMCSLayer{
		receiveFunc: func() (uint16, io.Reader, error) {
			buf := new(bytes.Buffer)
			
			// Security header with SEC_LICENSE_PKT flag
			_ = binary.Write(buf, binary.LittleEndian, uint32(0x0080))
			
			// Preamble
			_ = binary.Write(buf, binary.LittleEndian, uint8(0xFF)) // ERROR_ALERT
			_ = binary.Write(buf, binary.LittleEndian, uint8(0x02))
			_ = binary.Write(buf, binary.LittleEndian, uint16(20))
			
			// ValidClientMessage
			_ = binary.Write(buf, binary.LittleEndian, uint32(0x00000007)) // STATUS_VALID_CLIENT
			_ = binary.Write(buf, binary.LittleEndian, uint32(0x00000002)) // ST_NO_TRANSITION
			_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // BlobType
			_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // BlobLen
			
			return 1003, buf, nil
		},
	}

	client := &Client{
		mcsLayer:         mockMCS,
		selectedProtocol: pdu.NegotiationProtocolSSL,
	}

	err := client.licensing()
	require.NoError(t, err)
}

// Test licensing with NEW_LICENSE
func TestClient_licensing_NewLicense(t *testing.T) {
	mockMCS := &testMCSLayer{
		receiveFunc: func() (uint16, io.Reader, error) {
			buf := new(bytes.Buffer)
			
			// Security header
			_ = binary.Write(buf, binary.LittleEndian, uint32(0x0080))
			
			// Preamble with NEW_LICENSE
			_ = binary.Write(buf, binary.LittleEndian, uint8(0x03)) // NEW_LICENSE
			_ = binary.Write(buf, binary.LittleEndian, uint8(0x02))
			_ = binary.Write(buf, binary.LittleEndian, uint16(4))
			
			// Dummy data for ValidClientMessage
			_ = binary.Write(buf, binary.LittleEndian, uint32(0))
			_ = binary.Write(buf, binary.LittleEndian, uint32(0))
			_ = binary.Write(buf, binary.LittleEndian, uint16(0))
			_ = binary.Write(buf, binary.LittleEndian, uint16(0))
			
			return 1003, buf, nil
		},
	}

	client := &Client{
		mcsLayer:         mockMCS,
		selectedProtocol: pdu.NegotiationProtocolSSL,
	}

	err := client.licensing()
	require.NoError(t, err)
}
