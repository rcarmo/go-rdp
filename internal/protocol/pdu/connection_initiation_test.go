package pdu

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestClientConnectionRequestPDU_Serialize from MS-RDPBCGR Protocol examples 4.1.1.
// without TPKT header
func TestClientConnectionRequestPDU_Serialize(t *testing.T) {
	var req ClientConnectionRequest

	req.Cookie = "eltons"
	req.NegotiationRequest.RequestedProtocols = NegotiationProtocolRDP

	actual := req.Serialize()
	expected := []byte{
		0x43, 0x6f, 0x6f, 0x6b, 0x69, 0x65, 0x3a, 0x20, 0x6d, 0x73, 0x74, 0x73, 0x68, 0x61, 0x73, 0x68,
		0x3d, 0x65, 0x6c, 0x74, 0x6f, 0x6e, 0x73, 0x0d, 0x0a, 0x01, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00,
		0x00,
	}

	require.Equal(t, expected, actual)
}

// TestServerConnectionConfirmPDU_Deserialize from MS-RDPBCGR Protocol examples 4.1.2.
// without TPKT header
func TestServerConnectionConfirmPDU_Deserialize(t *testing.T) {
	var actual ServerConnectionConfirm

	expected := ServerConnectionConfirm{
		Type:   NegotiationTypeResponse,
		Flags:  0,
		length: 8,
		data:   uint32(NegotiationProtocolRDP),
	}

	input := bytes.NewBuffer([]byte{
		0x02, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00,
	})

	require.NoError(t, actual.Deserialize(input))
	require.Equal(t, expected, actual)
}

func TestNegotiationType_IsMethods(t *testing.T) {
	tests := []struct {
		name       string
		negType    NegotiationType
		isRequest  bool
		isResponse bool
		isFailure  bool
	}{
		{"Request", NegotiationTypeRequest, true, false, false},
		{"Response", NegotiationTypeResponse, false, true, false},
		{"Failure", NegotiationTypeFailure, false, false, true},
		{"Unknown", NegotiationType(0xFF), false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isRequest, tt.negType.IsRequest())
			require.Equal(t, tt.isResponse, tt.negType.IsResponse())
			require.Equal(t, tt.isFailure, tt.negType.IsFailure())
		})
	}
}

func TestNegotiationRequestFlag_IsMethods(t *testing.T) {
	tests := []struct {
		name                         string
		flag                         NegotiationRequestFlag
		isRestrictedAdmin            bool
		isRedirectedAuth             bool
		isCorrelationInfoPresent     bool
	}{
		{"None", NegotiationRequestFlag(0), false, false, false},
		{"RestrictedAdmin", NegReqFlagRestrictedAdminModeRequired, true, false, false},
		{"RedirectedAuth", NegReqFlagRedirectedAuthenticationModeRequired, false, true, false},
		{"CorrelationInfo", NegReqFlagCorrelationInfoPresent, false, false, true},
		{"Multiple", NegReqFlagRestrictedAdminModeRequired | NegReqFlagCorrelationInfoPresent, true, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isRestrictedAdmin, tt.flag.IsRestrictedAdminModeRequired())
			require.Equal(t, tt.isRedirectedAuth, tt.flag.IsRedirectedAuthenticationModeRequired())
			require.Equal(t, tt.isCorrelationInfoPresent, tt.flag.IsCorrelationInfoPresent())
		})
	}
}

func TestNegotiationProtocol_IsMethods(t *testing.T) {
	tests := []struct {
		name       string
		protocol   NegotiationProtocol
		isRDP      bool
		isSSL      bool
		isHybrid   bool
		isRDSTLS   bool
		isHybridEx bool
	}{
		{"RDP", NegotiationProtocolRDP, true, false, false, false, false},
		{"SSL", NegotiationProtocolSSL, false, true, false, false, false},
		{"Hybrid", NegotiationProtocolHybrid, false, false, true, false, false},
		{"RDSTLS", NegotiationProtocolRDSTLS, false, false, false, true, false},
		{"HybridEx", NegotiationProtocolHybridEx, false, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isRDP, tt.protocol.IsRDP())
			require.Equal(t, tt.isSSL, tt.protocol.IsSSL())
			require.Equal(t, tt.isHybrid, tt.protocol.IsHybrid())
			require.Equal(t, tt.isRDSTLS, tt.protocol.IsRDSTLS())
			require.Equal(t, tt.isHybridEx, tt.protocol.IsHybridEx())
		})
	}
}

func TestServerConnectionConfirm_SelectedProtocolAndFailureCode(t *testing.T) {
	t.Run("SelectedProtocol", func(t *testing.T) {
		confirm := ServerConnectionConfirm{
			Type: NegotiationTypeResponse,
			data: uint32(NegotiationProtocolSSL),
		}
		require.Equal(t, NegotiationProtocolSSL, confirm.SelectedProtocol())
	})

	t.Run("FailureCode", func(t *testing.T) {
		confirm := ServerConnectionConfirm{
			Type: NegotiationTypeFailure,
			data: 0x00000001, // SSL_REQUIRED_BY_SERVER
		}
		require.Equal(t, NegotiationFailureCode(0x00000001), confirm.FailureCode())
	})
}

func TestNegotiationResponseFlag_IsMethods(t *testing.T) {
	tests := []struct {
		name                    string
		flag                    NegotiationResponseFlag
		isExtendedClientData    bool
		isGFXProtocol           bool
		isRestrictedAdminMode   bool
		isRedirectedAuthMode    bool
	}{
		{"None", NegotiationResponseFlag(0), false, false, false, false},
		{"ExtendedClientData", NegotiationResponseFlagECDBSupported, true, false, false, false},
		{"GFXProtocol", NegotiationResponseFlagGFXSupported, false, true, false, false},
		{"RestrictedAdminMode", NegotiationResponseFlagAdminModeSupported, false, false, true, false},
		{"RedirectedAuthMode", NegotiationResponseFlagAuthModeSupported, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.isExtendedClientData, tt.flag.IsExtendedClientDataSupported())
			require.Equal(t, tt.isGFXProtocol, tt.flag.IsGFXProtocolSupported())
			require.Equal(t, tt.isRestrictedAdminMode, tt.flag.IsRestrictedAdminModeSupported())
			require.Equal(t, tt.isRedirectedAuthMode, tt.flag.IsRedirectedAuthModeSupported())
		})
	}
}

func TestNegotiationResponseFlag_String(t *testing.T) {
	flag := NegotiationResponseFlagECDBSupported
	str := flag.String()
	require.Contains(t, str, "EXTENDED_CLIENT_DATA_SUPPORTED")
}

func TestNegotiationFailureCode_String(t *testing.T) {
	tests := []struct {
		name     string
		code     NegotiationFailureCode
		contains string
	}{
		{"SSLRequiredByServer", NegotiationFailureCodeSSLRequired, "SSL_REQUIRED_BY_SERVER"},
		{"SSLNotAllowedByServer", NegotiationFailureCodeSSLNotAllowed, "SSL_NOT_ALLOWED_BY_SERVER"},
		{"SSLCertNotOnServer", NegotiationFailureCodeSSLCertNotOnServer, "SSL_CERT_NOT_ON_SERVER"},
		{"InconsistentFlags", NegotiationFailureCodeInconsistentFlags, "INCONSISTENT_FLAGS"},
		{"HybridRequiredByServer", NegotiationFailureCodeHybridRequired, "HYBRID_REQUIRED_BY_SERVER"},
		{"SSLWithUserAuthRequiredByServer", NegotiationFailureCodeSSLWithUserAuthRequired, "SSL_WITH_USER_AUTH_REQUIRED_BY_SERVER"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.code.String()
			require.Contains(t, result, tt.contains)
		})
	}
}

func TestServerConnectionConfirm_DeserializeFailure(t *testing.T) {
	var confirm ServerConnectionConfirm

	input := bytes.NewBuffer([]byte{
		0x03, 0x00, 0x08, 0x00, // Type = Failure, Flags = 0, Length = 8
		0x01, 0x00, 0x00, 0x00, // FailureCode = SSL_REQUIRED_BY_SERVER
	})

	require.NoError(t, confirm.Deserialize(input))
	require.Equal(t, NegotiationTypeFailure, confirm.Type)
	require.Equal(t, NegotiationFailureCodeSSLRequired, confirm.FailureCode())
}
