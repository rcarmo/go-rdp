package mcs

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockX224Conn implements x224Conn interface for testing
type mockX224Conn struct {
	sendErr    error
	sendCalled bool
	sendData   []byte

	receiveData io.Reader
	receiveErr  error
}

func (m *mockX224Conn) Send(pduData []byte) error {
	m.sendCalled = true
	m.sendData = pduData
	return m.sendErr
}

func (m *mockX224Conn) Receive() (io.Reader, error) {
	return m.receiveData, m.receiveErr
}

func TestNew(t *testing.T) {
	// Can't create with nil, but test the constructor exists
	require.NotNil(t, newWithConn(&mockX224Conn{}))
}

func TestProtocol_Send(t *testing.T) {
	testCases := []struct {
		name      string
		userID    uint16
		channelID uint16
		data      []byte
		sendErr   error
		wantErr   bool
	}{
		{
			name:      "successful send",
			userID:    1007,
			channelID: 1003,
			data:      []byte{0x01, 0x02, 0x03},
		},
		{
			name:      "send error",
			userID:    1007,
			channelID: 1003,
			data:      []byte{0x01},
			sendErr:   errors.New("connection closed"),
			wantErr:   true,
		},
		{
			name:      "empty data",
			userID:    1007,
			channelID: 1003,
			data:      []byte{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockX224Conn{sendErr: tc.sendErr}
			p := newWithConn(mock)

			err := p.Send(tc.userID, tc.channelID, tc.data)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.True(t, mock.sendCalled)
			require.NotEmpty(t, mock.sendData)
		})
	}
}

func TestProtocol_Receive(t *testing.T) {
	testCases := []struct {
		name        string
		receiveData []byte
		receiveErr  error
		wantChannel uint16
		wantErr     error
	}{
		{
			name:        "successful receive",
			receiveData: []byte{0x68, 0x00, 0x06, 0x03, 0xeb, 0x00, 0x04},
			wantChannel: 1003,
		},
		{
			name:       "receive error",
			receiveErr: errors.New("connection closed"),
			wantErr:    errors.New("connection closed"),
		},
		{
			name:        "disconnect ultimatum",
			receiveData: []byte{0x20, 0x80},
			wantErr:     ErrDisconnectUltimatum,
		},
		{
			name:        "unknown application",
			receiveData: []byte{0x28}, // attachUserRequest
			wantErr:     ErrUnknownDomainApplication,
		},
		{
			name:        "malformed data",
			receiveData: []byte{0x68}, // truncated
			wantErr:     io.EOF,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockX224Conn{
				receiveData: bytes.NewBuffer(tc.receiveData),
				receiveErr:  tc.receiveErr,
			}
			p := newWithConn(mock)

			channelID, _, err := p.Receive()

			if tc.wantErr != nil {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantChannel, channelID)
		})
	}
}

func TestProtocol_ErectDomain(t *testing.T) {
	testCases := []struct {
		name    string
		sendErr error
		wantErr bool
	}{
		{
			name: "successful erect domain",
		},
		{
			name:    "send error",
			sendErr: errors.New("connection closed"),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockX224Conn{sendErr: tc.sendErr}
			p := newWithConn(mock)

			err := p.ErectDomain()

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.True(t, mock.sendCalled)
		})
	}
}

func TestProtocol_AttachUser(t *testing.T) {
	testCases := []struct {
		name          string
		sendErr       error
		receiveData   []byte
		receiveErr    error
		wantInitiator uint16
		wantErr       bool
	}{
		{
			name:          "successful attach user",
			receiveData:   []byte{0x2e, 0x00, 0x00, 0x06},
			wantInitiator: 1007,
		},
		{
			name:    "send error",
			sendErr: errors.New("connection closed"),
			wantErr: true,
		},
		{
			name:       "receive error",
			receiveErr: errors.New("connection closed"),
			wantErr:    true,
		},
		{
			name:        "unsuccessful result",
			receiveData: []byte{0x2e, 0x01, 0x00, 0x06}, // Result = 1 (not successful)
			wantErr:     true,
		},
		{
			name:        "malformed response",
			receiveData: []byte{0x2e, 0x00}, // truncated
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockX224Conn{
				sendErr:     tc.sendErr,
				receiveData: bytes.NewBuffer(tc.receiveData),
				receiveErr:  tc.receiveErr,
			}
			p := newWithConn(mock)

			initiator, err := p.AttachUser()

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantInitiator, initiator)
		})
	}
}

// multiReceiveMock allows multiple receive calls with different responses
type multiReceiveMock struct {
	sendErr     error
	sendCalled  bool
	sendData    []byte
	sendCount   int
	receiveData []io.Reader
	receiveErrs []error
	receiveIdx  int
}

func (m *multiReceiveMock) Send(pduData []byte) error {
	m.sendCalled = true
	m.sendData = pduData
	m.sendCount++
	return m.sendErr
}

func (m *multiReceiveMock) Receive() (io.Reader, error) {
	if m.receiveIdx >= len(m.receiveData) {
		return nil, io.EOF
	}
	data := m.receiveData[m.receiveIdx]
	var err error
	if m.receiveIdx < len(m.receiveErrs) {
		err = m.receiveErrs[m.receiveIdx]
	}
	m.receiveIdx++
	return data, err
}

func TestProtocol_JoinChannels(t *testing.T) {
	testCases := []struct {
		name        string
		channelMap  map[string]uint16
		sendErr     error
		receiveData []io.Reader
		receiveErrs []error
		wantErr     bool
	}{
		{
			name:       "empty channel map",
			channelMap: map[string]uint16{},
		},
		{
			name:       "successful join single channel",
			channelMap: map[string]uint16{"user": 1007},
			receiveData: []io.Reader{
				bytes.NewBuffer([]byte{0x3e, 0x00, 0x00, 0x06, 0x03, 0xef, 0x03, 0xef}),
			},
		},
		{
			name: "successful join multiple channels",
			channelMap: map[string]uint16{
				"user":   1007,
				"io":     1003,
				"rdpdr":  1004,
			},
			receiveData: []io.Reader{
				bytes.NewBuffer([]byte{0x3e, 0x00, 0x00, 0x06, 0x03, 0xef, 0x03, 0xef}),
				bytes.NewBuffer([]byte{0x3e, 0x00, 0x00, 0x06, 0x03, 0xeb, 0x03, 0xeb}),
				bytes.NewBuffer([]byte{0x3e, 0x00, 0x00, 0x06, 0x03, 0xec, 0x03, 0xec}),
			},
		},
		{
			name:       "send error",
			channelMap: map[string]uint16{"user": 1007},
			sendErr:    errors.New("connection closed"),
			wantErr:    true,
		},
		{
			name:        "receive error",
			channelMap:  map[string]uint16{"user": 1007},
			receiveErrs: []error{errors.New("connection closed")},
			receiveData: []io.Reader{nil},
			wantErr:     true,
		},
		{
			name:       "unsuccessful result",
			channelMap: map[string]uint16{"user": 1007},
			receiveData: []io.Reader{
				bytes.NewBuffer([]byte{0x3e, 0x01, 0x00, 0x06, 0x03, 0xef, 0x03, 0xef}), // Result = 1
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &multiReceiveMock{
				sendErr:     tc.sendErr,
				receiveData: tc.receiveData,
				receiveErrs: tc.receiveErrs,
			}
			p := &Protocol{x224Conn: mock}

			err := p.JoinChannels(1007, tc.channelMap)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestProtocol_Disconnect(t *testing.T) {
	testCases := []struct {
		name    string
		sendErr error
		wantErr bool
	}{
		{
			name: "successful disconnect",
		},
		{
			name:    "send error",
			sendErr: errors.New("connection closed"),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockX224Conn{sendErr: tc.sendErr}
			p := newWithConn(mock)

			err := p.Disconnect()

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.True(t, mock.sendCalled)
			require.Equal(t, []byte{0x21, 0x80}, mock.sendData)
		})
	}
}

func TestProtocol_Connect(t *testing.T) {
	// Build a valid MCS connect response
	validConnectResponse := []byte{
		0x7f, 0x66, 0x82, 0x01, 0x45, 0x0a, 0x01, 0x00, 0x02,
		0x01, 0x00, 0x30, 0x1a, 0x02, 0x01, 0x22, 0x02, 0x01, 0x03, 0x02, 0x01, 0x00, 0x02, 0x01, 0x01,
		0x02, 0x01, 0x00, 0x02, 0x01, 0x01, 0x02, 0x03, 0x00, 0xff, 0xf8, 0x02, 0x01, 0x02, 0x04, 0x82,
		0x01, 0x1f, 0x00, 0x05, 0x00, 0x14, 0x7c, 0x00, 0x01, 0x2a, 0x14, 0x76, 0x0a, 0x01, 0x01, 0x00,
		0x01, 0xc0, 0x00, 0x4d, 0x63, 0x44, 0x6e, 0x81, 0x08, 0x01, 0x0c, 0x0c, 0x00, 0x04, 0x00, 0x08,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x0c, 0x10, 0x00, 0xeb, 0x03, 0x03, 0x00, 0xec, 0x03, 0xed,
		0x03, 0xee, 0x03, 0x00, 0x00, 0x02, 0x0c, 0xec, 0x00, 0x02, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00,
		0x00, 0x20, 0x00, 0x00, 0x00, 0xb8, 0x00, 0x00, 0x00, 0x10, 0x11, 0x77, 0x20, 0x30, 0x61, 0x0a,
		0x12, 0xe4, 0x34, 0xa1, 0x1e, 0xf2, 0xc3, 0x9f, 0x31, 0x7d, 0xa4, 0x5f, 0x01, 0x89, 0x34, 0x96,
		0xe0, 0xff, 0x11, 0x08, 0x69, 0x7f, 0x1a, 0xc3, 0xd2, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00, 0x5c, 0x00, 0x52, 0x53, 0x41, 0x31, 0x48, 0x00, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x00, 0x3f, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0xcb, 0x81, 0xfe,
		0xba, 0x6d, 0x61, 0xc3, 0x55, 0x05, 0xd5, 0x5f, 0x2e, 0x87, 0xf8, 0x71, 0x94, 0xd6, 0xf1, 0xa5,
		0xcb, 0xf1, 0x5f, 0x0c, 0x3d, 0xf8, 0x70, 0x02, 0x96, 0xc4, 0xfb, 0x9b, 0xc8, 0x3c, 0x2d, 0x55,
		0xae, 0xe8, 0xff, 0x32, 0x75, 0xea, 0x68, 0x79, 0xe5, 0xa2, 0x01, 0xfd, 0x31, 0xa0, 0xb1, 0x1f,
		0x55, 0xa6, 0x1f, 0xc1, 0xf6, 0xd1, 0x83, 0x88, 0x63, 0x26, 0x56, 0x12, 0xbc, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x08, 0x00, 0x48, 0x00, 0xe9, 0xe1, 0xd6, 0x28, 0x46, 0x8b, 0x4e,
		0xf5, 0x0a, 0xdf, 0xfd, 0xee, 0x21, 0x99, 0xac, 0xb4, 0xe1, 0x8f, 0x5f, 0x81, 0x57, 0x82, 0xef,
		0x9d, 0x96, 0x52, 0x63, 0x27, 0x18, 0x29, 0xdb, 0xb3, 0x4a, 0xfd, 0x9a, 0xda, 0x42, 0xad, 0xb5,
		0x69, 0x21, 0x89, 0x0e, 0x1d, 0xc0, 0x4c, 0x1a, 0xa8, 0xaa, 0x71, 0x3e, 0x0f, 0x54, 0xb9, 0x9a,
		0xe4, 0x99, 0x68, 0x3f, 0x6c, 0xd6, 0x76, 0x84, 0x61, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00,
	}

	// Build an unsuccessful connect response (Result = 1)
	unsuccessfulResponse := make([]byte, len(validConnectResponse))
	copy(unsuccessfulResponse, validConnectResponse)
	unsuccessfulResponse[7] = 0x01 // Change result from 0 to 1 (position is: 7f 66 82 01 45 0a 01 [00])

	testCases := []struct {
		name        string
		userData    []byte
		sendErr     error
		receiveData []byte
		receiveErr  error
		wantErr     bool
	}{
		{
			name:        "successful connect",
			userData:    []byte{0x01, 0x02, 0x03},
			receiveData: validConnectResponse,
		},
		{
			name:     "send error",
			userData: []byte{0x01, 0x02, 0x03},
			sendErr:  errors.New("connection closed"),
			wantErr:  true,
		},
		{
			name:       "receive error",
			userData:   []byte{0x01, 0x02, 0x03},
			receiveErr: errors.New("connection closed"),
			wantErr:    true,
		},
		{
			name:        "unsuccessful result",
			userData:    []byte{0x01, 0x02, 0x03},
			receiveData: unsuccessfulResponse,
			wantErr:     true,
		},
		{
			name:        "malformed response",
			userData:    []byte{0x01, 0x02, 0x03},
			receiveData: []byte{0x7f, 0x66}, // truncated
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockX224Conn{
				sendErr:     tc.sendErr,
				receiveData: bytes.NewBuffer(tc.receiveData),
				receiveErr:  tc.receiveErr,
			}
			p := newWithConn(mock)

			_, err := p.Connect(tc.userData)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestDomainPDUApplication_Values(t *testing.T) {
	// Test all DomainPDUApplication constants
	require.Equal(t, DomainPDUApplication(0), plumbDomainIndication)
	require.Equal(t, DomainPDUApplication(1), erectDomainRequest)
	require.Equal(t, DomainPDUApplication(2), mergeChannelsRequest)
	require.Equal(t, DomainPDUApplication(3), mergeChannelsConfirm)
	require.Equal(t, DomainPDUApplication(4), purgeChannelsIndication)
	require.Equal(t, DomainPDUApplication(5), mergeTokensRequest)
	require.Equal(t, DomainPDUApplication(6), mergeTokensConfirm)
	require.Equal(t, DomainPDUApplication(7), purgeTokensIndication)
	require.Equal(t, DomainPDUApplication(8), disconnectProviderUltimatum)
	require.Equal(t, DomainPDUApplication(9), rejectMCSPDUUltimatum)
	require.Equal(t, DomainPDUApplication(10), attachUserRequest)
	require.Equal(t, DomainPDUApplication(11), attachUserConfirm)
	require.Equal(t, DomainPDUApplication(12), detachUserRequest)
	require.Equal(t, DomainPDUApplication(13), detachUserIndication)
	require.Equal(t, DomainPDUApplication(14), channelJoinRequest)
	require.Equal(t, DomainPDUApplication(15), channelJoinConfirm)
	require.Equal(t, DomainPDUApplication(16), channelLeaveRequest)
	require.Equal(t, DomainPDUApplication(17), channelConveneRequest)
	require.Equal(t, DomainPDUApplication(18), channelConveneConfirm)
	require.Equal(t, DomainPDUApplication(19), channelDisbandRequest)
	require.Equal(t, DomainPDUApplication(20), channelDisbandIndication)
	require.Equal(t, DomainPDUApplication(21), channelAdmitRequest)
	require.Equal(t, DomainPDUApplication(22), channelAdmitIndication)
	require.Equal(t, DomainPDUApplication(23), channelExpelRequest)
	require.Equal(t, DomainPDUApplication(24), channelExpelIndication)
	require.Equal(t, DomainPDUApplication(25), SendDataRequest)
	require.Equal(t, DomainPDUApplication(26), SendDataIndication)
	require.Equal(t, DomainPDUApplication(27), uniformSendDataRequest)
	require.Equal(t, DomainPDUApplication(28), uniformSendDataIndication)
}
