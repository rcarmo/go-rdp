package rdp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/rcarmo/go-rdp/internal/protocol/audio"
	"github.com/rcarmo/go-rdp/internal/protocol/pdu"
)func TestBasicSettingsExchange_Success(t *testing.T) {
	client := &Client{
		selectedProtocol: pdu.NegotiationProtocolSSL,
		desktopWidth:     1920,
		desktopHeight:    1080,
		colorDepth:       24,
		channels:         []string{"rdpsnd", "cliprdr"},
		channelIDMap:     make(map[string]uint16),
	}

	// Create a server user data response
	serverUserData := createTestServerUserDataResponse(t)

	mockMCS := &MockMCSLayer{
		ConnectFunc: func(userData []byte) (io.Reader, error) {
			return bytes.NewReader(serverUserData), nil
		},
	}

	client.mcsLayer = mockMCS

	err := client.basicSettingsExchange()
	require.NoError(t, err)
	assert.False(t, client.skipChannelJoin)
}

// TestChannelConnection_Success tests the channel connection phase
func TestChannelConnection_Success(t *testing.T) {
	tests := []struct {
		name            string
		skipChannelJoin bool
		erectDomainErr  error
		attachUserErr   error
		joinChannelsErr error
		expectErr       bool
	}{
		{
			name:            "successful connection",
			skipChannelJoin: false,
			expectErr:       false,
		},
		{
			name:            "skip channel join",
			skipChannelJoin: true,
			expectErr:       false,
		},
		{
			name:           "erect domain error",
			erectDomainErr: errors.New("erect domain failed"),
			expectErr:      true,
		},
		{
			name:          "attach user error",
			attachUserErr: errors.New("attach user failed"),
			expectErr:     true,
		},
		{
			name:            "join channels error",
			joinChannelsErr: errors.New("join channels failed"),
			expectErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				skipChannelJoin: tt.skipChannelJoin,
				channelIDMap:    make(map[string]uint16),
			}

			mockMCS := &MockMCSLayer{
				ErectDomainFunc: func() error {
					return tt.erectDomainErr
				},
				AttachUserFunc: func() (uint16, error) {
					if tt.attachUserErr != nil {
						return 0, tt.attachUserErr
					}
					return 1001, nil
				},
				JoinChannelsFunc: func(userID uint16, channelIDMap map[string]uint16) error {
					return tt.joinChannelsErr
				},
			}

			client.mcsLayer = mockMCS

			err := client.channelConnection()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if !tt.skipChannelJoin {
					assert.Equal(t, uint16(1001), client.userID)
				}
			}
		})
	}
}

// TestSecureSettingsExchange_Protocols tests the secure settings exchange phase
func TestSecureSettingsExchange_Protocols(t *testing.T) {
	tests := []struct {
		name             string
		selectedProtocol pdu.NegotiationProtocol
		remoteApp        *RemoteApp
		sendErr          error
		expectErr        bool
	}{
		{
			name:             "SSL protocol",
			selectedProtocol: pdu.NegotiationProtocolSSL,
			expectErr:        false,
		},
		{
			name:             "Hybrid protocol",
			selectedProtocol: pdu.NegotiationProtocolHybrid,
			expectErr:        false,
		},
		{
			name:             "RDP protocol",
			selectedProtocol: pdu.NegotiationProtocolRDP,
			expectErr:        false,
		},
		{
			name:             "with remote app",
			selectedProtocol: pdu.NegotiationProtocolSSL,
			remoteApp:        &RemoteApp{App: "notepad"},
			expectErr:        false,
		},
		{
			name:             "send error",
			selectedProtocol: pdu.NegotiationProtocolSSL,
			sendErr:          errors.New("send failed"),
			expectErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				selectedProtocol: tt.selectedProtocol,
				domain:           "DOMAIN",
				username:         "user",
				password:         "pass",
				userID:           1001,
				channelIDMap:     map[string]uint16{"global": 1003},
				remoteApp:        tt.remoteApp,
			}

			mockMCS := &MockMCSLayer{
				SendFunc: func(userID, channelID uint16, data []byte) error {
					return tt.sendErr
				},
			}

			client.mcsLayer = mockMCS

			err := client.secureSettingsExchange()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLicensing_Scenarios tests the licensing phase
func TestLicensing_Scenarios(t *testing.T) {
	tests := []struct {
		name             string
		selectedProtocol pdu.NegotiationProtocol
		msgType          uint8
		errorCode        uint32
		stateTransition  uint32
		receiveErr       error
		expectErr        bool
	}{
		{
			name:             "receive error",
			selectedProtocol: pdu.NegotiationProtocolSSL,
			receiveErr:       errors.New("receive failed"),
			expectErr:        true,
		},
		{
			name:             "disconnect ultimatum",
			selectedProtocol: pdu.NegotiationProtocolSSL,
			receiveErr:       errors.New("disconnect ultimatum"),
			expectErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				selectedProtocol: tt.selectedProtocol,
			}

			mockMCS := &MockMCSLayer{
				ReceiveFunc: func() (uint16, io.Reader, error) {
					if tt.receiveErr != nil {
						return 0, nil, tt.receiveErr
					}
					buf := createTestLicenseResponse(t, tt.selectedProtocol, tt.msgType, tt.errorCode, tt.stateTransition)
					return 1003, bytes.NewReader(buf), nil
				},
			}

			client.mcsLayer = mockMCS

			err := client.licensing()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestInitChannels_Scenarios tests the channel initialization
func TestInitChannels_Scenarios(t *testing.T) {
	tests := []struct {
		name          string
		channels      []string
		channelIDs    []uint16
		mcsChannelID  uint16
		expectMapping map[string]uint16
	}{
		{
			name:         "two channels",
			channels:     []string{"rdpsnd", "cliprdr"},
			channelIDs:   []uint16{1004, 1005},
			mcsChannelID: 1003,
			expectMapping: map[string]uint16{
				"rdpsnd":  1004,
				"cliprdr": 1005,
				"global":  1003,
			},
		},
		{
			name:         "no channels",
			channels:     []string{},
			channelIDs:   []uint16{},
			mcsChannelID: 1003,
			expectMapping: map[string]uint16{
				"global": 1003,
			},
		},
		{
			name:         "more channels than IDs",
			channels:     []string{"rdpsnd", "cliprdr", "rdpdr"},
			channelIDs:   []uint16{1004, 1005},
			mcsChannelID: 1003,
			expectMapping: map[string]uint16{
				"rdpsnd":  1004,
				"cliprdr": 1005,
				"global":  1003,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				channels:     tt.channels,
				channelIDMap: nil, // Test nil initialization
			}

			serverNetworkData := &pdu.ServerNetworkData{
				MCSChannelId:   tt.mcsChannelID,
				ChannelIdArray: tt.channelIDs,
			}

			client.initChannels(serverNetworkData)

			for k, v := range tt.expectMapping {
				assert.Equal(t, v, client.channelIDMap[k], "channel %s should have ID %d", k, v)
			}
		})
	}
}

// Helper function to create a server user data response
func createTestServerUserDataResponse(t *testing.T) []byte {
	buf := new(bytes.Buffer)

	// Server Core Data header
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0C01)) // SC_CORE
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))    // Length
	_ = binary.Write(buf, binary.LittleEndian, uint32(0x00080004)) // Version
	_ = binary.Write(buf, binary.LittleEndian, uint32(1003))  // clientRequestedProtocols
	_ = binary.Write(buf, binary.LittleEndian, uint32(0))     // earlyCapabilityFlags

	// Server Network Data header
	_ = binary.Write(buf, binary.LittleEndian, uint16(0x0C03)) // SC_NET
	_ = binary.Write(buf, binary.LittleEndian, uint16(12))    // Length
	_ = binary.Write(buf, binary.LittleEndian, uint16(1003))  // MCSChannelId
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))     // channelCount
	_ = binary.Write(buf, binary.LittleEndian, uint16(1004))  // channel 1
	_ = binary.Write(buf, binary.LittleEndian, uint16(1005))  // channel 2

	return buf.Bytes()
}

// Helper function to create a license response
func createTestLicenseResponse(t *testing.T, protocol pdu.NegotiationProtocol, msgType uint8, errorCode, stateTransition uint32) []byte {
	buf := new(bytes.Buffer)

	// Security header (only for non-enhanced security)
	if !protocol.IsSSL() && !protocol.IsHybrid() {
		_ = binary.Write(buf, binary.LittleEndian, uint32(0x0080)) // SEC_LICENSE_PKT
	}

	// Preamble
	_ = binary.Write(buf, binary.LittleEndian, msgType)
	_ = binary.Write(buf, binary.LittleEndian, uint8(0x02))  // flags
	_ = binary.Write(buf, binary.LittleEndian, uint16(20))   // wMsgSize

	// ValidClientMessage
	_ = binary.Write(buf, binary.LittleEndian, errorCode)
	_ = binary.Write(buf, binary.LittleEndian, stateTransition)
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // wBlobType
	_ = binary.Write(buf, binary.LittleEndian, uint16(0)) // wBlobLen

	return buf.Bytes()
}
