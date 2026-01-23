// Package rdpemt implements the RDP Multitransport Extension Protocol (MS-RDPEMT).
// This protocol enables UDP-based transport negotiation for RDP connections.
package rdpemt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Protocol constants from [MS-RDPEMT] Section 2.2
const (
	// Action types for tunnel PDUs
	ActionCreateRequest  uint8 = 0x00
	ActionCreateResponse uint8 = 0x01
	ActionData           uint8 = 0x02

	// Requested protocol flags for multitransport request
	// [MS-RDPEMT] Section 2.2.2.1
	ProtocolUDPFECReliable uint16 = 0x0001 // Reliable UDP with FEC
	ProtocolUDPFECLossy    uint16 = 0x0002 // Lossy UDP with FEC (for audio/video)

	// HRESULT values for multitransport response
	// [MS-RDPBCGR] Section 2.2.15.2
	HResultSuccess  uint32 = 0x00000000 // S_OK
	HResultNoMem    uint32 = 0x80000002 // E_OUTOFMEMORY
	HResultNotFound uint32 = 0x80000006 // E_NOTFOUND (transport not available)
	HResultAbort    uint32 = 0x80004004 // E_ABORT (client declines)

	// Security cookie length
	CookieLength     = 16
	CookieHashLength = 32

	// Minimum PDU sizes
	MinRequestSize  = 24 // 4 + 2 + 2 + 16
	MinResponseSize = 8  // 4 + 4

	// Tunnel header size
	TunnelHeaderSize = 4 // Action(1) + Flags(1) + PayloadLength(2)
)

// Errors
var (
	ErrInvalidLength   = errors.New("rdpemt: invalid PDU length")
	ErrUnknownAction   = errors.New("rdpemt: unknown action type")
	ErrInvalidProtocol = errors.New("rdpemt: invalid protocol flags")
)

// MultitransportRequest represents the Server Initiate Multitransport Request PDU.
// [MS-RDPBCGR] Section 2.2.15.1
type MultitransportRequest struct {
	RequestID         uint32   // Unique identifier for this request
	RequestedProtocol uint16   // Protocol flags (ProtocolUDPFECReliable or ProtocolUDPFECLossy)
	Reserved          uint16   // Reserved, must be 0 (but may contain extension info)
	SecurityCookie    [16]byte // Random cookie for tunnel binding
}

// Deserialize parses a MultitransportRequest from bytes.
func (r *MultitransportRequest) Deserialize(data []byte) error {
	if len(data) < MinRequestSize {
		return fmt.Errorf("%w: need %d bytes, got %d", ErrInvalidLength, MinRequestSize, len(data))
	}

	buf := bytes.NewReader(data)
	binary.Read(buf, binary.LittleEndian, &r.RequestID)
	binary.Read(buf, binary.LittleEndian, &r.RequestedProtocol)
	binary.Read(buf, binary.LittleEndian, &r.Reserved)
	buf.Read(r.SecurityCookie[:])

	return nil
}

// Serialize encodes a MultitransportRequest to bytes.
func (r *MultitransportRequest) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, MinRequestSize))
	binary.Write(buf, binary.LittleEndian, r.RequestID)
	binary.Write(buf, binary.LittleEndian, r.RequestedProtocol)
	binary.Write(buf, binary.LittleEndian, r.Reserved)
	buf.Write(r.SecurityCookie[:])
	return buf.Bytes(), nil
}

// IsReliable returns true if the request is for reliable UDP transport.
func (r *MultitransportRequest) IsReliable() bool {
	return r.RequestedProtocol&ProtocolUDPFECReliable != 0
}

// IsLossy returns true if the request is for lossy UDP transport.
func (r *MultitransportRequest) IsLossy() bool {
	return r.RequestedProtocol&ProtocolUDPFECLossy != 0
}

// MultitransportResponse represents the Client Initiate Multitransport Response PDU.
// [MS-RDPBCGR] Section 2.2.15.2
type MultitransportResponse struct {
	RequestID uint32 // Must match the RequestID from the request
	HResult   uint32 // Result code (HResultSuccess or error)
}

// Deserialize parses a MultitransportResponse from bytes.
func (r *MultitransportResponse) Deserialize(data []byte) error {
	if len(data) < MinResponseSize {
		return fmt.Errorf("%w: need %d bytes, got %d", ErrInvalidLength, MinResponseSize, len(data))
	}

	buf := bytes.NewReader(data)
	binary.Read(buf, binary.LittleEndian, &r.RequestID)
	binary.Read(buf, binary.LittleEndian, &r.HResult)
	return nil
}

// Serialize encodes a MultitransportResponse to bytes.
func (r *MultitransportResponse) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, MinResponseSize))
	binary.Write(buf, binary.LittleEndian, r.RequestID)
	binary.Write(buf, binary.LittleEndian, r.HResult)
	return buf.Bytes(), nil
}

// IsSuccess returns true if the response indicates success.
func (r *MultitransportResponse) IsSuccess() bool {
	return r.HResult == HResultSuccess
}

// NewDeclineResponse creates a response declining the multitransport request.
func NewDeclineResponse(requestID uint32) *MultitransportResponse {
	return &MultitransportResponse{
		RequestID: requestID,
		HResult:   HResultAbort,
	}
}

// NewSuccessResponse creates a response accepting the multitransport request.
func NewSuccessResponse(requestID uint32) *MultitransportResponse {
	return &MultitransportResponse{
		RequestID: requestID,
		HResult:   HResultSuccess,
	}
}

// TunnelHeader represents the RDP_TUNNEL_HEADER structure.
// [MS-RDPEMT] Section 2.2.2.1
type TunnelHeader struct {
	Action        uint8  // ActionCreateRequest, ActionCreateResponse, or ActionData
	Flags         uint8  // Reserved flags
	PayloadLength uint16 // Length of payload following header
}

// Deserialize parses a TunnelHeader from bytes.
func (h *TunnelHeader) Deserialize(data []byte) error {
	if len(data) < TunnelHeaderSize {
		return fmt.Errorf("%w: need %d bytes, got %d", ErrInvalidLength, TunnelHeaderSize, len(data))
	}

	h.Action = data[0]
	h.Flags = data[1]
	h.PayloadLength = binary.LittleEndian.Uint16(data[2:4])
	return nil
}

// Serialize encodes a TunnelHeader to bytes.
func (h *TunnelHeader) Serialize() ([]byte, error) {
	buf := make([]byte, TunnelHeaderSize)
	buf[0] = h.Action
	buf[1] = h.Flags
	binary.LittleEndian.PutUint16(buf[2:4], h.PayloadLength)
	return buf, nil
}

// TunnelCreateRequest represents the RDP_TUNNEL_CREATEREQUEST structure.
// [MS-RDPEMT] Section 2.2.2.2
type TunnelCreateRequest struct {
	RequestID      uint32   // Request ID from the MultitransportRequest
	Reserved       uint32   // Reserved, must be 0
	SecurityCookie [16]byte // Cookie from the MultitransportRequest
}

// Size returns the serialized size of the structure.
func (r *TunnelCreateRequest) Size() int {
	return 4 + 4 + 16 // RequestID + Reserved + SecurityCookie
}

// Deserialize parses a TunnelCreateRequest from bytes.
func (r *TunnelCreateRequest) Deserialize(data []byte) error {
	if len(data) < r.Size() {
		return fmt.Errorf("%w: need %d bytes, got %d", ErrInvalidLength, r.Size(), len(data))
	}

	buf := bytes.NewReader(data)
	binary.Read(buf, binary.LittleEndian, &r.RequestID)
	binary.Read(buf, binary.LittleEndian, &r.Reserved)
	buf.Read(r.SecurityCookie[:])
	return nil
}

// Serialize encodes a TunnelCreateRequest to bytes.
func (r *TunnelCreateRequest) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, r.Size()))
	binary.Write(buf, binary.LittleEndian, r.RequestID)
	binary.Write(buf, binary.LittleEndian, r.Reserved)
	buf.Write(r.SecurityCookie[:])
	return buf.Bytes(), nil
}

// TunnelCreateResponse represents the RDP_TUNNEL_CREATERESPONSE structure.
// [MS-RDPEMT] Section 2.2.2.3
type TunnelCreateResponse struct {
	HResult uint32 // Result code
}

// Size returns the serialized size of the structure.
func (r *TunnelCreateResponse) Size() int {
	return 4
}

// Deserialize parses a TunnelCreateResponse from bytes.
func (r *TunnelCreateResponse) Deserialize(data []byte) error {
	if len(data) < r.Size() {
		return fmt.Errorf("%w: need %d bytes, got %d", ErrInvalidLength, r.Size(), len(data))
	}

	r.HResult = binary.LittleEndian.Uint32(data[:4])
	return nil
}

// Serialize encodes a TunnelCreateResponse to bytes.
func (r *TunnelCreateResponse) Serialize() ([]byte, error) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, r.HResult)
	return buf, nil
}

// TunnelDataPDU wraps data sent over the tunnel.
// [MS-RDPEMT] Section 2.2.2.4
type TunnelDataPDU struct {
	Header TunnelHeader
	Data   []byte // Wrapped RDP data
}

// Deserialize parses a TunnelDataPDU from bytes.
func (p *TunnelDataPDU) Deserialize(data []byte) error {
	if err := p.Header.Deserialize(data); err != nil {
		return err
	}

	if p.Header.Action != ActionData {
		return fmt.Errorf("%w: expected Data action, got %d", ErrUnknownAction, p.Header.Action)
	}

	payloadStart := TunnelHeaderSize
	payloadEnd := payloadStart + int(p.Header.PayloadLength)

	if len(data) < payloadEnd {
		return fmt.Errorf("%w: payload truncated", ErrInvalidLength)
	}

	p.Data = make([]byte, p.Header.PayloadLength)
	copy(p.Data, data[payloadStart:payloadEnd])
	return nil
}

// Serialize encodes a TunnelDataPDU to bytes.
func (p *TunnelDataPDU) Serialize() ([]byte, error) {
	p.Header.Action = ActionData
	p.Header.PayloadLength = uint16(len(p.Data))

	headerBytes, _ := p.Header.Serialize()
	result := make([]byte, 0, len(headerBytes)+len(p.Data))
	result = append(result, headerBytes...)
	result = append(result, p.Data...)
	return result, nil
}

// ParseTunnelPDU parses any tunnel PDU and returns the action type and payload.
func ParseTunnelPDU(data []byte) (action uint8, payload []byte, err error) {
	var header TunnelHeader
	if err := header.Deserialize(data); err != nil {
		return 0, nil, err
	}

	payloadStart := TunnelHeaderSize
	payloadEnd := payloadStart + int(header.PayloadLength)

	if len(data) < payloadEnd {
		return 0, nil, fmt.Errorf("%w: payload truncated", ErrInvalidLength)
	}

	return header.Action, data[payloadStart:payloadEnd], nil
}

// HResultString returns a human-readable description of an HRESULT code.
func HResultString(hr uint32) string {
	switch hr {
	case HResultSuccess:
		return "S_OK"
	case HResultNoMem:
		return "E_OUTOFMEMORY"
	case HResultNotFound:
		return "E_NOTFOUND"
	case HResultAbort:
		return "E_ABORT"
	default:
		return fmt.Sprintf("0x%08X", hr)
	}
}

// ProtocolString returns a human-readable description of protocol flags.
func ProtocolString(proto uint16) string {
	var parts []string
	if proto&ProtocolUDPFECReliable != 0 {
		parts = append(parts, "UDP-FEC-Reliable")
	}
	if proto&ProtocolUDPFECLossy != 0 {
		parts = append(parts, "UDP-FEC-Lossy")
	}
	if len(parts) == 0 {
		return "None"
	}
	return fmt.Sprintf("%v", parts)
}
