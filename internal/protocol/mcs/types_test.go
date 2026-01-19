package mcs

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomainParameters_Serialize(t *testing.T) {
	testCases := []struct {
		name     string
		params   domainParameters
		expected []byte
	}{
		{
			name: "target parameters",
			params: domainParameters{
				maxChannelIds:   34,
				maxUserIds:      2,
				maxTokenIds:     0,
				numPriorities:   1,
				minThroughput:   0,
				maxHeight:       1,
				maxMCSPDUsize:   65535,
				protocolVersion: 2,
			},
			expected: []byte{
				0x02, 0x01, 0x22, // maxChannelIds = 34
				0x02, 0x01, 0x02, // maxUserIds = 2
				0x02, 0x01, 0x00, // maxTokenIds = 0
				0x02, 0x01, 0x01, // numPriorities = 1
				0x02, 0x01, 0x00, // minThroughput = 0
				0x02, 0x01, 0x01, // maxHeight = 1
				0x02, 0x02, 0xff, 0xff, // maxMCSPDUsize = 65535
				0x02, 0x01, 0x02, // protocolVersion = 2
			},
		},
		{
			name: "minimum parameters",
			params: domainParameters{
				maxChannelIds:   1,
				maxUserIds:      1,
				maxTokenIds:     1,
				numPriorities:   1,
				minThroughput:   0,
				maxHeight:       1,
				maxMCSPDUsize:   1056,
				protocolVersion: 2,
			},
			expected: []byte{
				0x02, 0x01, 0x01, // maxChannelIds = 1
				0x02, 0x01, 0x01, // maxUserIds = 1
				0x02, 0x01, 0x01, // maxTokenIds = 1
				0x02, 0x01, 0x01, // numPriorities = 1
				0x02, 0x01, 0x00, // minThroughput = 0
				0x02, 0x01, 0x01, // maxHeight = 1
				0x02, 0x02, 0x04, 0x20, // maxMCSPDUsize = 1056
				0x02, 0x01, 0x02, // protocolVersion = 2
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.params.Serialize()
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestDomainParameters_Deserialize(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected domainParameters
		wantErr  bool
	}{
		{
			name: "valid parameters",
			input: []byte{
				0x02, 0x01, 0x22, // maxChannelIds = 34
				0x02, 0x01, 0x03, // maxUserIds = 3
				0x02, 0x01, 0x00, // maxTokenIds = 0
				0x02, 0x01, 0x01, // numPriorities = 1
				0x02, 0x01, 0x00, // minThroughput = 0
				0x02, 0x01, 0x01, // maxHeight = 1
				0x02, 0x03, 0x00, 0xff, 0xf8, // maxMCSPDUsize = 65528
				0x02, 0x01, 0x02, // protocolVersion = 2
			},
			expected: domainParameters{
				maxChannelIds:   34,
				maxUserIds:      3,
				maxTokenIds:     0,
				numPriorities:   1,
				minThroughput:   0,
				maxHeight:       1,
				maxMCSPDUsize:   65528,
				protocolVersion: 2,
			},
		},
		{
			name:    "truncated maxChannelIds",
			input:   []byte{0x02},
			wantErr: true,
		},
		{
			name: "truncated maxUserIds",
			input: []byte{
				0x02, 0x01, 0x22,
			},
			wantErr: true,
		},
		{
			name: "truncated maxTokenIds",
			input: []byte{
				0x02, 0x01, 0x22,
				0x02, 0x01, 0x03,
			},
			wantErr: true,
		},
		{
			name: "truncated numPriorities",
			input: []byte{
				0x02, 0x01, 0x22,
				0x02, 0x01, 0x03,
				0x02, 0x01, 0x00,
			},
			wantErr: true,
		},
		{
			name: "truncated minThroughput",
			input: []byte{
				0x02, 0x01, 0x22,
				0x02, 0x01, 0x03,
				0x02, 0x01, 0x00,
				0x02, 0x01, 0x01,
			},
			wantErr: true,
		},
		{
			name: "truncated maxHeight",
			input: []byte{
				0x02, 0x01, 0x22,
				0x02, 0x01, 0x03,
				0x02, 0x01, 0x00,
				0x02, 0x01, 0x01,
				0x02, 0x01, 0x00,
			},
			wantErr: true,
		},
		{
			name: "truncated maxMCSPDUsize",
			input: []byte{
				0x02, 0x01, 0x22,
				0x02, 0x01, 0x03,
				0x02, 0x01, 0x00,
				0x02, 0x01, 0x01,
				0x02, 0x01, 0x00,
				0x02, 0x01, 0x01,
			},
			wantErr: true,
		},
		{
			name: "truncated protocolVersion",
			input: []byte{
				0x02, 0x01, 0x22,
				0x02, 0x01, 0x03,
				0x02, 0x01, 0x00,
				0x02, 0x01, 0x01,
				0x02, 0x01, 0x00,
				0x02, 0x01, 0x01,
				0x02, 0x02, 0xff, 0xf8,
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var actual domainParameters
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

func TestResultTypes(t *testing.T) {
	// Test that result type constants are correctly defined
	require.Equal(t, uint8(0), RTSuccessful)
	require.Equal(t, uint8(1), RTDomainMerging)
	require.Equal(t, uint8(2), RTDomainNotHierarchical)
	require.Equal(t, uint8(3), RTNoSuchChannel)
	require.Equal(t, uint8(4), RTNoSuchDomain)
	require.Equal(t, uint8(5), RTNoSuchUser)
	require.Equal(t, uint8(6), RTNotAdmitted)
	require.Equal(t, uint8(7), RTOtherUserId)
	require.Equal(t, uint8(8), RTParametersUnacceptable)
	require.Equal(t, uint8(9), RTTokenNotAvailable)
	require.Equal(t, uint8(10), RTTokenNotPossessed)
	require.Equal(t, uint8(11), RTTooManyChannels)
	require.Equal(t, uint8(12), RTTooManyTokens)
	require.Equal(t, uint8(13), RTTooManyUsers)
	require.Equal(t, uint8(14), RTUnspecifiedFailure)
	require.Equal(t, uint8(15), RTUserRejected)
}

func TestReasonTypes(t *testing.T) {
	// Test that reason type constants are correctly defined
	require.Equal(t, uint8(0), RNDomainDisconnected)
	require.Equal(t, uint8(1), RNProviderInitiated)
	require.Equal(t, uint8(2), RNTokenPurged)
	require.Equal(t, uint8(3), RNUserRequested)
	require.Equal(t, uint8(4), RNChannelPurged)
}
