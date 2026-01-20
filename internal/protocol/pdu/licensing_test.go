package pdu

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLicensingBinaryBlob_Deserialize(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantType uint16
		wantLen  uint16
		wantData []byte
		wantErr  bool
	}{
		{
			name:     "empty blob",
			input:    []byte{0x01, 0x00, 0x00, 0x00},
			wantType: 1,
			wantLen:  0,
			wantData: nil,
		},
		{
			name:     "blob with data",
			input:    []byte{0x02, 0x00, 0x04, 0x00, 0xDE, 0xAD, 0xBE, 0xEF},
			wantType: 2,
			wantLen:  4,
			wantData: []byte{0xDE, 0xAD, 0xBE, 0xEF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blob := &LicensingBinaryBlob{}
			err := blob.Deserialize(bytes.NewReader(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantType, blob.BlobType)
			require.Equal(t, tt.wantLen, blob.BlobLen)
			require.Equal(t, tt.wantData, blob.BlobData)
		})
	}
}

func TestLicensingErrorMessage_Deserialize(t *testing.T) {
	// Error code (4) + State transition (4) + blob type (2) + blob len (2)
	input := []byte{
		0x07, 0x00, 0x00, 0x00, // ErrorCode = 7 (STATUS_VALID_CLIENT)
		0x02, 0x00, 0x00, 0x00, // StateTransition = 2 (ST_NO_TRANSITION)
		0x00, 0x00, // BlobType
		0x00, 0x00, // BlobLen = 0
	}

	msg := &LicensingErrorMessage{}
	err := msg.Deserialize(bytes.NewReader(input))
	require.NoError(t, err)
	require.Equal(t, uint32(7), msg.ErrorCode)
	require.Equal(t, uint32(2), msg.StateTransition)
}

func TestLicensingPreamble_Deserialize(t *testing.T) {
	input := []byte{
		0xFF, // MsgType = 0xFF (ERROR_ALERT)
		0x03, // Flags = 0x03
		0x10, 0x00, // MsgSize = 16
	}

	preamble := &LicensingPreamble{}
	err := preamble.Deserialize(bytes.NewReader(input))
	require.NoError(t, err)
	require.Equal(t, uint8(0xFF), preamble.MsgType)
	require.Equal(t, uint8(0x03), preamble.Flags)
	require.Equal(t, uint16(16), preamble.MsgSize)
}

func TestServerLicenseError_Deserialize(t *testing.T) {
	// Security header (4 bytes: flags + padding) + preamble + error message
	input := []byte{
		// Security header: SEC_LICENSE_PKT = 0x0080
		0x80, 0x00, 0x00, 0x00,
		// Preamble
		0xFF, 0x03, 0x10, 0x00,
		// Error message
		0x07, 0x00, 0x00, 0x00, // ErrorCode
		0x02, 0x00, 0x00, 0x00, // StateTransition
		0x00, 0x00, // BlobType
		0x00, 0x00, // BlobLen
	}

	pdu := &ServerLicenseError{}
	err := pdu.Deserialize(bytes.NewReader(input), true)
	require.NoError(t, err)
	require.Equal(t, uint8(0xFF), pdu.Preamble.MsgType)
	require.Equal(t, uint32(7), pdu.ValidClientMessage.ErrorCode)
}

func TestServerLicenseError_Deserialize_InvalidHeader(t *testing.T) {
	// Security header without SEC_LICENSE_PKT flag
	input := []byte{
		0x00, 0x00, 0x00, 0x00, // No SEC_LICENSE_PKT
	}

	pdu := &ServerLicenseError{}
	err := pdu.Deserialize(bytes.NewReader(input), true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "bad license header")
}
