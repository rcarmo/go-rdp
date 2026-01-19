package mcs

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnectPDU_Serialize(t *testing.T) {
	userData := []byte{0x01, 0x02, 0x03, 0x04}
	initialPDU := NewClientMCSConnectInitial(userData)

	pdu := ConnectPDU{
		Application:          connectInitial,
		ClientConnectInitial: initialPDU,
	}

	result := pdu.Serialize()
	// Verify it starts with BER application tag for connectInitial (101 = 0x65)
	require.True(t, len(result) > 0)
	require.Equal(t, uint8(0x7f), result[0])
	require.Equal(t, uint8(0x65), result[1])
}

func TestConnectPDU_Deserialize_Errors(t *testing.T) {
	testCases := []struct {
		name    string
		input   []byte
		wantErr error
	}{
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: io.EOF,
		},
		{
			name:    "truncated application tag",
			input:   []byte{0x7f},
			wantErr: io.EOF,
		},
		{
			name:    "unknown application",
			input:   []byte{0x7f, 0x67, 0x00},
			wantErr: ErrUnknownConnectApplication,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var pdu ConnectPDU
			err := pdu.Deserialize(bytes.NewBuffer(tc.input))
			require.Error(t, err)
			if tc.wantErr != nil {
				require.True(t, errors.Is(err, tc.wantErr), "got: %v, want: %v", err, tc.wantErr)
			}
		})
	}
}

func TestConnectPDUApplication_Values(t *testing.T) {
	require.Equal(t, ConnectPDUApplication(101), connectInitial)
	require.Equal(t, ConnectPDUApplication(102), connectResponse)
	require.Equal(t, ConnectPDUApplication(103), connectAdditional)
	require.Equal(t, ConnectPDUApplication(104), connectResult)
}

func TestNewClientMCSConnectInitial(t *testing.T) {
	userData := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	pdu := NewClientMCSConnectInitial(userData)

	require.NotNil(t, pdu)
	require.Equal(t, []byte{0x01}, pdu.calledDomainSelector)
	require.Equal(t, []byte{0x01}, pdu.callingDomainSelector)
	require.True(t, pdu.upwardFlag)

	// Verify target parameters
	require.Equal(t, 34, pdu.targetParameters.maxChannelIds)
	require.Equal(t, 2, pdu.targetParameters.maxUserIds)
	require.Equal(t, 0, pdu.targetParameters.maxTokenIds)
	require.Equal(t, 1, pdu.targetParameters.numPriorities)
	require.Equal(t, 0, pdu.targetParameters.minThroughput)
	require.Equal(t, 1, pdu.targetParameters.maxHeight)
	require.Equal(t, 65535, pdu.targetParameters.maxMCSPDUsize)
	require.Equal(t, 2, pdu.targetParameters.protocolVersion)

	// Verify minimum parameters
	require.Equal(t, 1, pdu.minimumParameters.maxChannelIds)
	require.Equal(t, 1, pdu.minimumParameters.maxUserIds)
	require.Equal(t, 1, pdu.minimumParameters.maxTokenIds)
	require.Equal(t, 1, pdu.minimumParameters.numPriorities)
	require.Equal(t, 0, pdu.minimumParameters.minThroughput)
	require.Equal(t, 1, pdu.minimumParameters.maxHeight)
	require.Equal(t, 1056, pdu.minimumParameters.maxMCSPDUsize)
	require.Equal(t, 2, pdu.minimumParameters.protocolVersion)

	// Verify maximum parameters
	require.Equal(t, 65535, pdu.maximumParameters.maxChannelIds)
	require.Equal(t, 65535, pdu.maximumParameters.maxUserIds)
	require.Equal(t, 65535, pdu.maximumParameters.maxTokenIds)
	require.Equal(t, 1, pdu.maximumParameters.numPriorities)
	require.Equal(t, 0, pdu.maximumParameters.minThroughput)
	require.Equal(t, 1, pdu.maximumParameters.maxHeight)
	require.Equal(t, 65535, pdu.maximumParameters.maxMCSPDUsize)
	require.Equal(t, 2, pdu.maximumParameters.protocolVersion)

	require.NotNil(t, pdu.userData)
}

func TestClientConnectInitial_Serialize(t *testing.T) {
	userData := []byte{0x01, 0x02}
	pdu := NewClientMCSConnectInitial(userData)
	result := pdu.Serialize()

	// Should produce valid BER-encoded output
	require.True(t, len(result) > 0)
	// First byte should be octet string tag (0x04) for calledDomainSelector
	require.Equal(t, uint8(0x04), result[0])
}

func TestServerConnectResponse_Deserialize_Errors(t *testing.T) {
	testCases := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "empty input",
			input:   []byte{},
			wantErr: true,
		},
		{
			name:    "truncated result",
			input:   []byte{0x0a},
			wantErr: true,
		},
		{
			name:    "truncated calledConnectId",
			input:   []byte{0x0a, 0x01, 0x00},
			wantErr: true,
		},
		{
			name: "bad BER tag for sequence",
			input: []byte{
				0x0a, 0x01, 0x00, // result = 0
				0x02, 0x01, 0x00, // calledConnectId = 0
				0x00, 0x00, // wrong tag
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var pdu ServerConnectResponse
			err := pdu.Deserialize(bytes.NewBuffer(tc.input))

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
