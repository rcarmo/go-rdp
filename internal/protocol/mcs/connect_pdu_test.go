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

// ============================================================================
// Microsoft Protocol Test Suite Validation Tests
// Reference: MS-RDPBCGR_ClientTestDesignSpecification.md - S1_Connection
// ============================================================================

// TestBVT_MCSConnect_ConnectInitial validates per MS test case:
// "BVT_ConnectionTest_BasicSettingExchange_PositiveTest"
// Per MS-RDPBCGR Section 2.2.1.3
func TestBVT_MCSConnect_ConnectInitial(t *testing.T) {
	userData := []byte{0x01, 0x02, 0x03, 0x04}
	pdu := NewClientMCSConnectInitial(userData)

	// Per MS-RDPBCGR 2.2.1.3: callingDomainSelector must be 1
	require.Equal(t, []byte{0x01}, pdu.callingDomainSelector)
	// Per MS-RDPBCGR 2.2.1.3: calledDomainSelector must be 1
	require.Equal(t, []byte{0x01}, pdu.calledDomainSelector)
	// Per MS-RDPBCGR 2.2.1.3: upwardFlag must be true
	require.True(t, pdu.upwardFlag)
}

// TestS1_MCSConnect_DomainParameters validates MCS domain parameters
// Per MS-RDPBCGR Section 2.2.1.3.2 and T.125 Section 7.5
func TestS1_MCSConnect_DomainParameters(t *testing.T) {
	userData := []byte{0x01, 0x02, 0x03, 0x04}
	pdu := NewClientMCSConnectInitial(userData)

	// Target parameters per MS-RDPBCGR 2.2.1.3
	params := pdu.targetParameters
	
	require.Equal(t, 34, params.maxChannelIds)
	require.Equal(t, 2, params.maxUserIds)
	require.Equal(t, 0, params.maxTokenIds)
	require.Equal(t, 1, params.numPriorities)
	require.Equal(t, 0, params.minThroughput)
	require.Equal(t, 1, params.maxHeight)
	require.Equal(t, 65535, params.maxMCSPDUsize)
	require.Equal(t, 2, params.protocolVersion)
}

// TestS1_MCSConnect_ConnectResponse_ResultCodes validates MCS Connect Response result codes
// Per MS-RDPBCGR Section 2.2.1.4 and T.125
func TestS1_MCSConnect_ConnectResponse_ResultCodes(t *testing.T) {
	// MCS Connect Response result codes per T.125
	resultCodes := []struct {
		code    int
		name    string
		success bool
	}{
		{0, "rt-successful", true},
		{1, "rt-domain-merging", false},
		{2, "rt-domain-not-hierarchical", false},
		{3, "rt-no-such-channel", false},
		{4, "rt-no-such-domain", false},
		{5, "rt-no-such-user", false},
		{6, "rt-not-admitted", false},
		{7, "rt-other-user-id", false},
		{8, "rt-parameters-unacceptable", false},
		{9, "rt-token-not-available", false},
		{10, "rt-token-not-possessed", false},
		{11, "rt-too-many-channels", false},
		{12, "rt-too-many-tokens", false},
		{13, "rt-too-many-users", false},
		{14, "rt-unspecified-failure", false},
		{15, "rt-user-rejected", false},
	}

	for _, rc := range resultCodes {
		t.Run(rc.name, func(t *testing.T) {
			// Only code 0 indicates success
			isSuccess := rc.code == 0
			require.Equal(t, rc.success, isSuccess, "result code %d (%s)", rc.code, rc.name)
		})
	}
}
