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

	// Tunnel header sizes
	// Per [MS-RDPEMT] Section 2.2.1.1:
	// - Byte 0: Action (4 bits) + Flags (4 bits) combined
	// - Bytes 1-2: PayloadLength (16-bit)
	// - Byte 3: HeaderLength (8-bit)
	TunnelHeaderMinSize = 4 // ActionFlags(1) + PayloadLength(2) + HeaderLength(1)
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
	binary.Read(buf, binary.LittleEndian, &r.RequestID)         // #nosec G104 -- in-memory buffer
	binary.Read(buf, binary.LittleEndian, &r.RequestedProtocol) // #nosec G104 -- in-memory buffer
	binary.Read(buf, binary.LittleEndian, &r.Reserved)          // #nosec G104 -- in-memory buffer
	buf.Read(r.SecurityCookie[:])                               // #nosec G104 -- in-memory buffer

	return nil
}

// Serialize encodes a MultitransportRequest to bytes.
func (r *MultitransportRequest) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, MinRequestSize))
	binary.Write(buf, binary.LittleEndian, r.RequestID)         // #nosec G104 -- in-memory buffer
	binary.Write(buf, binary.LittleEndian, r.RequestedProtocol) // #nosec G104 -- in-memory buffer
	binary.Write(buf, binary.LittleEndian, r.Reserved)          // #nosec G104 -- in-memory buffer
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
	binary.Read(buf, binary.LittleEndian, &r.RequestID) // #nosec G104 -- in-memory buffer
	binary.Read(buf, binary.LittleEndian, &r.HResult)   // #nosec G104 -- in-memory buffer
	return nil
}

// Serialize encodes a MultitransportResponse to bytes.
func (r *MultitransportResponse) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, MinResponseSize))
	binary.Write(buf, binary.LittleEndian, r.RequestID) // #nosec G104 -- in-memory buffer
	binary.Write(buf, binary.LittleEndian, r.HResult)   // #nosec G104 -- in-memory buffer
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
// [MS-RDPEMT] Section 2.2.1.1
//
// Wire format:
//
//	Byte 0:   | Action (4 bits) | Flags (4 bits) |
//	Bytes 1-2: PayloadLength (16-bit little-endian)
//	Byte 3:    HeaderLength (8-bit, min 4)
//	Bytes 4+:  SubHeaders (variable, if HeaderLength > 4)
type TunnelHeader struct {
	Action        uint8  // Lower 4 bits: ActionCreateRequest, ActionCreateResponse, or ActionData
	Flags         uint8  // Upper 4 bits: Reserved, must be 0
	PayloadLength uint16 // Length of payload following header (not including header)
	HeaderLength  uint8  // Total header length (min 4, includes SubHeaders if present)
	SubHeaders    []byte // Optional sub-headers (present if HeaderLength > 4)
}

// Size returns the actual wire size of this header.
func (h *TunnelHeader) Size() int {
	if h.HeaderLength < TunnelHeaderMinSize {
		return TunnelHeaderMinSize
	}
	return int(h.HeaderLength)
}

// Deserialize parses a TunnelHeader from bytes.
func (h *TunnelHeader) Deserialize(data []byte) error {
	if len(data) < TunnelHeaderMinSize {
		return fmt.Errorf("%w: need %d bytes, got %d", ErrInvalidLength, TunnelHeaderMinSize, len(data))
	}

	// Byte 0: Action (low nibble) + Flags (high nibble)
	h.Action = data[0] & 0x0F
	h.Flags = (data[0] >> 4) & 0x0F
	h.PayloadLength = binary.LittleEndian.Uint16(data[1:3])
	h.HeaderLength = data[3]

	// Validate HeaderLength
	if h.HeaderLength < TunnelHeaderMinSize {
		h.HeaderLength = TunnelHeaderMinSize // Treat invalid as minimum
	}

	// Read SubHeaders if present
	if h.HeaderLength > TunnelHeaderMinSize {
		subHeaderLen := int(h.HeaderLength) - TunnelHeaderMinSize
		if len(data) < int(h.HeaderLength) {
			return fmt.Errorf("%w: header truncated, need %d bytes, got %d", ErrInvalidLength, h.HeaderLength, len(data))
		}
		h.SubHeaders = make([]byte, subHeaderLen)
		copy(h.SubHeaders, data[TunnelHeaderMinSize:h.HeaderLength])
	}

	return nil
}

// Serialize encodes a TunnelHeader to bytes.
func (h *TunnelHeader) Serialize() ([]byte, error) {
	// Calculate header length
	headerLen := TunnelHeaderMinSize + len(h.SubHeaders)
	if headerLen > 255 {
		return nil, fmt.Errorf("header too large: %d bytes", headerLen)
	}

	buf := make([]byte, headerLen)
	// Byte 0: Action (low nibble) + Flags (high nibble)
	buf[0] = (h.Flags << 4) | (h.Action & 0x0F)
	binary.LittleEndian.PutUint16(buf[1:3], h.PayloadLength)
	buf[3] = uint8(headerLen) // #nosec G115

	// Copy SubHeaders if present
	if len(h.SubHeaders) > 0 {
		copy(buf[TunnelHeaderMinSize:], h.SubHeaders)
	}

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
	binary.Read(buf, binary.LittleEndian, &r.RequestID) // #nosec G104 -- in-memory buffer
	binary.Read(buf, binary.LittleEndian, &r.Reserved)  // #nosec G104 -- in-memory buffer
	buf.Read(r.SecurityCookie[:])                        // #nosec G104 -- in-memory buffer
	return nil
}

// Serialize encodes a TunnelCreateRequest to bytes.
func (r *TunnelCreateRequest) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, r.Size()))
	binary.Write(buf, binary.LittleEndian, r.RequestID) // #nosec G104 -- in-memory buffer
	binary.Write(buf, binary.LittleEndian, r.Reserved)  // #nosec G104 -- in-memory buffer
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
// [MS-RDPEMT] Section 2.2.2.3
type TunnelDataPDU struct {
	Header TunnelHeader
	Data   []byte // Wrapped RDP data (HigherLayerData)
}

// Deserialize parses a TunnelDataPDU from bytes.
func (p *TunnelDataPDU) Deserialize(data []byte) error {
	if err := p.Header.Deserialize(data); err != nil {
		return err
	}

	if p.Header.Action != ActionData {
		return fmt.Errorf("%w: expected Data action, got %d", ErrUnknownAction, p.Header.Action)
	}

	payloadStart := p.Header.Size()
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
	p.Header.Flags = 0 // Must be zero per spec
	p.Header.PayloadLength = uint16(len(p.Data)) // #nosec G115

	headerBytes, err := p.Header.Serialize()
	if err != nil {
		return nil, err
	}

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

	payloadStart := header.Size()
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
