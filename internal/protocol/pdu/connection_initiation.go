package pdu

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
)

// NegotiationType represents the type field in RDP negotiation structures (MS-RDPBCGR 2.2.1.1).
type NegotiationType uint8

const (
	// NegotiationTypeRequest TYPE_RDP_NEG_REQ
	NegotiationTypeRequest NegotiationType = 0x01

	// NegotiationTypeResponse TYPE_RDP_NEG_RSP
	NegotiationTypeResponse NegotiationType = 0x02

	// NegotiationTypeFailure TYPE_RDP_NEG_FAILURE
	NegotiationTypeFailure NegotiationType = 0x03
)

// IsRequest returns true if the type is a negotiation request.
func (t NegotiationType) IsRequest() bool {
	return t == NegotiationTypeRequest
}

// IsResponse returns true if the type is a negotiation response.
func (t NegotiationType) IsResponse() bool {
	return t == NegotiationTypeResponse
}

// IsFailure returns true if the type is a negotiation failure.
func (t NegotiationType) IsFailure() bool {
	return t == NegotiationTypeFailure
}

// NegotiationRequestFlag Protocol flags.
type NegotiationRequestFlag uint8

const (
	// NegReqFlagRestrictedAdminModeRequired RESTRICTED_ADMIN_MODE_REQUIRED
	NegReqFlagRestrictedAdminModeRequired NegotiationRequestFlag = 0x01

	// NegReqFlagRedirectedAuthenticationModeRequired REDIRECTED_AUTHENTICATION_MODE_REQUIRED
	NegReqFlagRedirectedAuthenticationModeRequired NegotiationRequestFlag = 0x02

	// NegReqFlagCorrelationInfoPresent CORRELATION_INFO_PRESENT
	NegReqFlagCorrelationInfoPresent NegotiationRequestFlag = 0x08
)

// IsRestrictedAdminModeRequired returns true if restricted admin mode is required.
func (f NegotiationRequestFlag) IsRestrictedAdminModeRequired() bool {
	return f&NegReqFlagRestrictedAdminModeRequired == NegReqFlagRestrictedAdminModeRequired
}

// IsRedirectedAuthenticationModeRequired returns true if redirected authentication mode is required.
func (f NegotiationRequestFlag) IsRedirectedAuthenticationModeRequired() bool {
	return f&NegReqFlagRedirectedAuthenticationModeRequired == NegReqFlagRedirectedAuthenticationModeRequired
}

// IsCorrelationInfoPresent returns true if correlation info is present.
func (f NegotiationRequestFlag) IsCorrelationInfoPresent() bool {
	return f&NegReqFlagCorrelationInfoPresent == NegReqFlagCorrelationInfoPresent
}

// NegotiationProtocol Supported security protocol.
type NegotiationProtocol uint32

const (
	// NegotiationProtocolRDP PROTOCOL_RDP
	NegotiationProtocolRDP NegotiationProtocol = 0x00000000

	// NegotiationProtocolSSL PROTOCOL_SSL
	NegotiationProtocolSSL NegotiationProtocol = 0x00000001

	// NegotiationProtocolHybrid PROTOCOL_HYBRID
	NegotiationProtocolHybrid NegotiationProtocol = 0x00000002

	// NegotiationProtocolRDSTLS PROTOCOL_RDSTLS
	NegotiationProtocolRDSTLS NegotiationProtocol = 0x00000004

	// NegotiationProtocolHybridEx PROTOCOL_HYBRID_EX
	NegotiationProtocolHybridEx NegotiationProtocol = 0x00000008
)

// IsRDP returns true if the protocol is standard RDP security.
func (p NegotiationProtocol) IsRDP() bool {
	return p == NegotiationProtocolRDP
}

// IsSSL returns true if the protocol is TLS security.
func (p NegotiationProtocol) IsSSL() bool {
	return p == NegotiationProtocolSSL
}

// IsHybrid returns true if the protocol is CredSSP (TLS + NLA).
func (p NegotiationProtocol) IsHybrid() bool {
	return p == NegotiationProtocolHybrid
}

// IsRDSTLS returns true if the protocol is RDSTLS.
func (p NegotiationProtocol) IsRDSTLS() bool {
	return p == NegotiationProtocolRDSTLS
}

// IsHybridEx returns true if the protocol is CredSSP with Early User Auth.
func (p NegotiationProtocol) IsHybridEx() bool {
	return p == NegotiationProtocolHybridEx
}

// NegotiationRequest RDP Negotiation Request (RDP_NEG_REQ).
type NegotiationRequest struct {
	Flags              NegotiationRequestFlag // Protocol flags
	RequestedProtocols NegotiationProtocol    // supported security protocols
}

// Serialize encodes the negotiation request to wire format.
func (r NegotiationRequest) Serialize() []byte {
	const negReqLen = uint16(8)

	buf := bytes.NewBuffer(make([]byte, 0, negReqLen))

	buf.Write([]byte{
		byte(NegotiationTypeRequest), // type TYPE_RDP_NEG_REQ
		byte(r.Flags),                // flags
	})

	// length (always 8 bytes)
	_ = binary.Write(buf, binary.LittleEndian, negReqLen)

	// requestedProtocols
	_ = binary.Write(buf, binary.LittleEndian, r.RequestedProtocols)

	return buf.Bytes()
}

// CorrelationInfo RDP Correlation Info (RDP_NEG_CORRELATION_INFO).
type CorrelationInfo struct {
	correlationID []byte
}

// SetCorrelationID sets the correlation ID and validates it per MS-RDPBCGR 2.2.1.1.2.
func (i CorrelationInfo) SetCorrelationID(correlationID []byte) error {
	if len(correlationID) != 16 {
		return ErrInvalidCorrelationID
	}

	// The first byte in the array SHOULD NOT have a value of 0x00 or 0xF4
	if correlationID[0] == 0x00 || correlationID[0] == 0xF4 {
		return ErrInvalidCorrelationID
	}

	// value 0x0D SHOULD NOT be contained in any of the bytes
	for _, b := range correlationID {
		if b == 0x0D {
			return ErrInvalidCorrelationID
		}
	}

	return nil
}

// Serialize encodes the correlation info to wire format.
func (i CorrelationInfo) Serialize() []byte {
	const corrInfoLen = uint16(36)

	buf := bytes.NewBuffer(make([]byte, 0, corrInfoLen))

	buf.Write([]byte{
		0x06, // type TYPE_RDP_CORRELATION_INFO
		0x00, // flags
	})

	// length (always 36 bytes)
	_ = binary.Write(buf, binary.LittleEndian, corrInfoLen)

	// correlationId
	if i.correlationID == nil {
		buf.Write(make([]byte, 16))
	} else {
		buf.Write(i.correlationID)
	}

	// reserved
	buf.Write(make([]byte, 16))

	return buf.Bytes()
}

// NegotiationResponseFlag RDP Negotiation Response flags
type NegotiationResponseFlag uint8

const (
	// NegotiationResponseFlagECDBSupported EXTENDED_CLIENT_DATA_SUPPORTED
	NegotiationResponseFlagECDBSupported NegotiationResponseFlag = 0x01

	// NegotiationResponseFlagGFXSupported DYNVC_GFX_PROTOCOL_SUPPORTED
	NegotiationResponseFlagGFXSupported NegotiationResponseFlag = 0x02

	// NegotiationResponseFlagAdminModeSupported RESTRICTED_ADMIN_MODE_SUPPORTED
	NegotiationResponseFlagAdminModeSupported NegotiationResponseFlag = 0x08

	// NegotiationResponseFlagAuthModeSupported REDIRECTED_AUTHENTICATION_MODE_SUPPORTED
	NegotiationResponseFlagAuthModeSupported NegotiationResponseFlag = 0x10
)

// IsExtendedClientDataSupported returns true if extended client data is supported.
func (f NegotiationResponseFlag) IsExtendedClientDataSupported() bool {
	return f&NegotiationResponseFlagECDBSupported == NegotiationResponseFlagECDBSupported
}

// IsGFXProtocolSupported returns true if GFX protocol is supported.
func (f NegotiationResponseFlag) IsGFXProtocolSupported() bool {
	return f&NegotiationResponseFlagGFXSupported == NegotiationResponseFlagGFXSupported
}

// IsRestrictedAdminModeSupported returns true if restricted admin mode is supported.
func (f NegotiationResponseFlag) IsRestrictedAdminModeSupported() bool {
	return f&NegotiationResponseFlagAdminModeSupported == NegotiationResponseFlagAdminModeSupported
}

// IsRedirectedAuthModeSupported returns true if redirected auth mode is supported.
func (f NegotiationResponseFlag) IsRedirectedAuthModeSupported() bool {
	return f&NegotiationResponseFlagAuthModeSupported == NegotiationResponseFlagAuthModeSupported
}

// String returns a human-readable representation of the response flags.
func (f NegotiationResponseFlag) String() string {
	var features []string

	switch {
	case f.IsExtendedClientDataSupported():
		features = append(features, "EXTENDED_CLIENT_DATA_SUPPORTED")
	case f.IsGFXProtocolSupported():
		features = append(features, "DYNVC_GFX_PROTOCOL_SUPPORTED")
	case f.IsRestrictedAdminModeSupported():
		features = append(features, "RESTRICTED_ADMIN_MODE_SUPPORTED")
	case f.IsRedirectedAuthModeSupported():
		features = append(features, "REDIRECTED_AUTHENTICATION_MODE_SUPPORTED")
	}

	return strings.Join(features, ", ")
}

// NegotiationFailureCode RDP Negotiation Failure failureCode
type NegotiationFailureCode uint32

const (
	// NegotiationFailureCodeSSLRequired SSL_REQUIRED_BY_SERVER
	NegotiationFailureCodeSSLRequired NegotiationFailureCode = 0x00000001

	// NegotiationFailureCodeSSLNotAllowed SSL_NOT_ALLOWED_BY_SERVER
	NegotiationFailureCodeSSLNotAllowed NegotiationFailureCode = 0x00000002

	// NegotiationFailureCodeSSLCertNotOnServer SSL_CERT_NOT_ON_SERVER
	NegotiationFailureCodeSSLCertNotOnServer NegotiationFailureCode = 0x00000003

	// NegotiationFailureCodeInconsistentFlags INCONSISTENT_FLAGS
	NegotiationFailureCodeInconsistentFlags NegotiationFailureCode = 0x00000004

	// NegotiationFailureCodeHybridRequired HYBRID_REQUIRED_BY_SERVER
	NegotiationFailureCodeHybridRequired NegotiationFailureCode = 0x00000005

	// NegotiationFailureCodeSSLWithUserAuthRequired SSL_WITH_USER_AUTH_REQUIRED_BY_SERVER
	NegotiationFailureCodeSSLWithUserAuthRequired NegotiationFailureCode = 0x00000006
)

// NegotiationFailureCodeMap maps failure codes to their string representations.
var NegotiationFailureCodeMap = map[NegotiationFailureCode]string{
	NegotiationFailureCodeSSLRequired:             "SSL_REQUIRED_BY_SERVER",
	NegotiationFailureCodeSSLNotAllowed:           "SSL_NOT_ALLOWED_BY_SERVER",
	NegotiationFailureCodeSSLCertNotOnServer:      "SSL_CERT_NOT_ON_SERVER",
	NegotiationFailureCodeInconsistentFlags:       "INCONSISTENT_FLAGS",
	NegotiationFailureCodeHybridRequired:          "HYBRID_REQUIRED_BY_SERVER",
	NegotiationFailureCodeSSLWithUserAuthRequired: "SSL_WITH_USER_AUTH_REQUIRED_BY_SERVER",
}

// String returns the string representation of the failure code.
func (c NegotiationFailureCode) String() string {
	return NegotiationFailureCodeMap[c]
}

// ClientConnectionRequest Client X.224 Connection Request PDU
type ClientConnectionRequest struct {
	RoutingToken       string // one of RoutingToken or Cookie ending CR+LF
	Cookie             string
	NegotiationRequest NegotiationRequest // RDP Negotiation Request
	CorrelationInfo    CorrelationInfo    // Correlation Info
}

// Serialize encodes the connection request to wire format.
func (pdu *ClientConnectionRequest) Serialize() []byte {
	const (
		CRLF         = "\r\n"
		cookieHeader = "Cookie: mstshash="
	)

	buf := new(bytes.Buffer)

	// routingToken or cookie
	if pdu.RoutingToken != "" {
		buf.WriteString(strings.Trim(pdu.RoutingToken, CRLF) + CRLF)
	} else if pdu.Cookie != "" {
		buf.WriteString(cookieHeader + strings.Trim(pdu.Cookie, CRLF) + CRLF)
	}

	// rdpNegReq
	buf.Write(pdu.NegotiationRequest.Serialize())

	// rdpCorrelationInfo
	if pdu.NegotiationRequest.Flags.IsCorrelationInfoPresent() {
		buf.Write(pdu.CorrelationInfo.Serialize())
	}

	return buf.Bytes()
}

// ServerConnectionConfirm represents the Server X.224 Connection Confirm PDU (MS-RDPBCGR 2.2.1.2).
type ServerConnectionConfirm struct {
	Type   NegotiationType
	Flags  NegotiationResponseFlag // RDP Negotiation Response flags
	length uint16
	data   uint32 // RDP Negotiation Response selectedProtocol or RDP Negotiation Failure failureCode
}

// SelectedProtocol returns the selected security protocol from the response.
func (pdu *ServerConnectionConfirm) SelectedProtocol() NegotiationProtocol {
	return NegotiationProtocol(pdu.data)
}

// FailureCode returns the failure code if the negotiation failed.
func (pdu *ServerConnectionConfirm) FailureCode() NegotiationFailureCode {
	return NegotiationFailureCode(pdu.data)
}

// Deserialize decodes the connection confirm from wire format.
func (pdu *ServerConnectionConfirm) Deserialize(wire io.Reader) error {
	err := binary.Read(wire, binary.LittleEndian, &pdu.Type)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &pdu.Flags)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &pdu.length)
	if err != nil {
		return err
	}

	err = binary.Read(wire, binary.LittleEndian, &pdu.data)
	if err != nil {
		return err
	}

	return nil
}
