package mcs

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomainPDU_Serialize(t *testing.T) {
	testCases := []struct {
		name     string
		pdu      DomainPDU
		expected []byte
	}{
		{
			name: "attachUserRequest",
			pdu: DomainPDU{
				Application:             attachUserRequest,
				ClientAttachUserRequest: &ClientAttachUserRequest{},
			},
			expected: []byte{0x28},
		},
		{
			name: "erectDomainRequest",
			pdu: DomainPDU{
				Application:              erectDomainRequest,
				ClientErectDomainRequest: &ClientErectDomainRequest{},
			},
			expected: []byte{0x04, 0x01, 0x00, 0x01, 0x00},
		},
		{
			name: "channelJoinRequest",
			pdu: DomainPDU{
				Application: channelJoinRequest,
				ClientChannelJoinRequest: &ClientChannelJoinRequest{
					Initiator: 1007,
					ChannelId: 1003,
				},
			},
			expected: []byte{0x38, 0x00, 0x06, 0x03, 0xeb},
		},
		{
			name: "SendDataRequest",
			pdu: DomainPDU{
				Application: SendDataRequest,
				ClientSendDataRequest: &ClientSendDataRequest{
					Initiator: 1007,
					ChannelId: 1003,
					Data:      []byte{0x01, 0x02, 0x03},
				},
			},
			expected: []byte{0x64, 0x00, 0x06, 0x03, 0xeb, 0x70, 0x03, 0x01, 0x02, 0x03},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.pdu.Serialize()
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestDomainPDU_Deserialize(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected DomainPDU
		wantErr  error
	}{
		{
			name:  "attachUserConfirm",
			input: []byte{0x2e, 0x00, 0x00, 0x06},
			expected: DomainPDU{
				Application: attachUserConfirm,
				ServerAttachUserConfirm: &ServerAttachUserConfirm{
					Result:    0x00,
					Initiator: 1007,
				},
			},
		},
		{
			name:  "channelJoinConfirm",
			input: []byte{0x3e, 0x00, 0x00, 0x06, 0x03, 0xeb, 0x03, 0xeb},
			expected: DomainPDU{
				Application: channelJoinConfirm,
				ServerChannelJoinConfirm: &ServerChannelJoinConfirm{
					Result:    0x00,
					Initiator: 1007,
					Requested: 1003,
					ChannelId: 1003,
				},
			},
		},
		{
			name:  "SendDataIndication",
			input: []byte{0x68, 0x00, 0x06, 0x03, 0xeb, 0x00, 0x03},
			expected: DomainPDU{
				Application: SendDataIndication,
				ServerSendDataIndication: &ServerSendDataIndication{
					Initiator: 1007,
					ChannelId: 1003,
				},
			},
		},
		{
			name:  "SendDataRequest deserialize",
			input: []byte{0x64, 0x00, 0x06, 0x03, 0xeb, 0x70, 0x03},
			expected: DomainPDU{
				Application: SendDataRequest,
				ClientSendDataRequest: &ClientSendDataRequest{
					Initiator: 1007,
					ChannelId: 1003,
				},
			},
		},
		{
			name:    "disconnectProviderUltimatum",
			input:   []byte{0x20, 0x80},
			wantErr: ErrDisconnectUltimatum,
		},
		{
			name:    "unknown application",
			input:   []byte{0x00},
			wantErr: ErrUnknownDomainApplication,
		},
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: io.EOF,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var actual DomainPDU
			err := actual.Deserialize(bytes.NewBuffer(tc.input))

			if tc.wantErr != nil {
				require.Error(t, err)
				require.True(t, errors.Is(err, tc.wantErr), "got: %v, want: %v", err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestClientSendDataRequest_Serialize(t *testing.T) {
	testCases := []struct {
		name     string
		req      ClientSendDataRequest
		expected []byte
	}{
		{
			name: "basic data",
			req: ClientSendDataRequest{
				Initiator: 1007,
				ChannelId: 1003,
				Data:      []byte{0xDE, 0xAD, 0xBE, 0xEF},
			},
			expected: []byte{0x00, 0x06, 0x03, 0xeb, 0x70, 0x04, 0xDE, 0xAD, 0xBE, 0xEF},
		},
		{
			name: "empty data",
			req: ClientSendDataRequest{
				Initiator: 1007,
				ChannelId: 1003,
				Data:      []byte{},
			},
			expected: []byte{0x00, 0x06, 0x03, 0xeb, 0x70, 0x00},
		},
		{
			name: "different channel",
			req: ClientSendDataRequest{
				Initiator: 1007,
				ChannelId: 1004,
				Data:      []byte{0x01},
			},
			expected: []byte{0x00, 0x06, 0x03, 0xec, 0x70, 0x01, 0x01},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.req.Serialize()
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestClientSendDataRequest_Deserialize(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected ClientSendDataRequest
		wantErr  bool
	}{
		{
			name:  "valid request",
			input: []byte{0x00, 0x06, 0x03, 0xeb, 0x70, 0x04},
			expected: ClientSendDataRequest{
				Initiator: 1007,
				ChannelId: 1003,
			},
		},
		{
			name:    "truncated initiator",
			input:   []byte{0x00},
			wantErr: true,
		},
		{
			name:    "truncated channel",
			input:   []byte{0x00, 0x06},
			wantErr: true,
		},
		{
			name:    "missing magic",
			input:   []byte{0x00, 0x06, 0x03, 0xeb},
			wantErr: true,
		},
		{
			name:    "missing length",
			input:   []byte{0x00, 0x06, 0x03, 0xeb, 0x70},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var actual ClientSendDataRequest
			err := actual.Deserialize(bytes.NewBuffer(tc.input))

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestServerSendDataIndication_Deserialize(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected ServerSendDataIndication
		wantErr  bool
	}{
		{
			name:  "valid indication",
			input: []byte{0x00, 0x06, 0x03, 0xeb, 0x00, 0x04},
			expected: ServerSendDataIndication{
				Initiator: 1007,
				ChannelId: 1003,
			},
		},
		{
			name:  "different channel",
			input: []byte{0x00, 0x06, 0x03, 0xec, 0x00, 0x10},
			expected: ServerSendDataIndication{
				Initiator: 1007,
				ChannelId: 1004,
			},
		},
		{
			name:    "truncated initiator",
			input:   []byte{0x00},
			wantErr: true,
		},
		{
			name:    "truncated channel",
			input:   []byte{0x00, 0x06},
			wantErr: true,
		},
		{
			name:    "missing enumerates",
			input:   []byte{0x00, 0x06, 0x03, 0xeb},
			wantErr: true,
		},
		{
			name:    "missing length",
			input:   []byte{0x00, 0x06, 0x03, 0xeb, 0x00},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var actual ServerSendDataIndication
			err := actual.Deserialize(bytes.NewBuffer(tc.input))

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestServerAttachUserConfirm_Deserialize(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected ServerAttachUserConfirm
		wantErr  bool
	}{
		{
			name:  "successful confirm",
			input: []byte{0x00, 0x00, 0x06},
			expected: ServerAttachUserConfirm{
				Result:    0x00,
				Initiator: 1007,
			},
		},
		{
			name:  "different initiator",
			input: []byte{0x00, 0x00, 0x0A},
			expected: ServerAttachUserConfirm{
				Result:    0x00,
				Initiator: 1011,
			},
		},
		{
			name:    "truncated result",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "truncated initiator",
			input:   []byte{0x00},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var actual ServerAttachUserConfirm
			err := actual.Deserialize(bytes.NewBuffer(tc.input))

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestServerChannelJoinConfirm_Deserialize(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected ServerChannelJoinConfirm
		wantErr  bool
	}{
		{
			name:  "full confirm with channel id",
			input: []byte{0x00, 0x00, 0x06, 0x03, 0xef, 0x03, 0xef},
			expected: ServerChannelJoinConfirm{
				Result:    0x00,
				Initiator: 1007,
				Requested: 1007,
				ChannelId: 1007,
			},
		},
		{
			name:  "confirm without optional channel id (EOF)",
			input: []byte{0x00, 0x00, 0x06, 0x03, 0xef},
			expected: ServerChannelJoinConfirm{
				Result:    0x00,
				Initiator: 1007,
				Requested: 1007,
				ChannelId: 0,
			},
		},
		{
			name:    "truncated result",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "truncated initiator",
			input:   []byte{0x00},
			wantErr: true,
		},
		{
			name:    "truncated requested",
			input:   []byte{0x00, 0x00, 0x06},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var actual ServerChannelJoinConfirm
			err := actual.Deserialize(bytes.NewBuffer(tc.input))

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestClientChannelJoinRequest_Serialize(t *testing.T) {
	testCases := []struct {
		name     string
		req      ClientChannelJoinRequest
		expected []byte
	}{
		{
			name: "channel 1003",
			req: ClientChannelJoinRequest{
				Initiator: 1007,
				ChannelId: 1003,
			},
			expected: []byte{0x00, 0x06, 0x03, 0xeb},
		},
		{
			name: "channel 1007",
			req: ClientChannelJoinRequest{
				Initiator: 1007,
				ChannelId: 1007,
			},
			expected: []byte{0x00, 0x06, 0x03, 0xef},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.req.Serialize()
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestClientAttachUserRequest_Serialize(t *testing.T) {
	req := ClientAttachUserRequest{}
	result := req.Serialize()
	require.Nil(t, result)
}

func TestClientErectDomainRequest_Serialize(t *testing.T) {
	req := ClientErectDomainRequest{}
	result := req.Serialize()
	require.Equal(t, []byte{0x01, 0x00, 0x01, 0x00}, result)
}
