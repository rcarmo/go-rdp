package rdpemt

import (
	"bytes"
	"testing"
)

func TestMultitransportRequest_Serialize(t *testing.T) {
	req := &MultitransportRequest{
		RequestID:         0x12345678,
		RequestedProtocol: ProtocolUDPFECReliable,
		Reserved:          0,
	}
	copy(req.SecurityCookie[:], []byte("0123456789ABCDEF"))

	data, err := req.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	if len(data) != MinRequestSize {
		t.Errorf("Expected %d bytes, got %d", MinRequestSize, len(data))
	}

	// Verify deserialization round-trip
	req2 := &MultitransportRequest{}
	if err := req2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if req2.RequestID != req.RequestID {
		t.Errorf("RequestID mismatch: %d vs %d", req2.RequestID, req.RequestID)
	}
	if req2.RequestedProtocol != req.RequestedProtocol {
		t.Errorf("Protocol mismatch: %d vs %d", req2.RequestedProtocol, req.RequestedProtocol)
	}
	if !bytes.Equal(req2.SecurityCookie[:], req.SecurityCookie[:]) {
		t.Error("SecurityCookie mismatch")
	}
}

func TestMultitransportRequest_Deserialize_TooShort(t *testing.T) {
	req := &MultitransportRequest{}
	err := req.Deserialize(make([]byte, 10))
	if err == nil {
		t.Error("Expected error for short data")
	}
}

func TestMultitransportRequest_ProtocolFlags(t *testing.T) {
	tests := []struct {
		name       string
		protocol   uint16
		isReliable bool
		isLossy    bool
	}{
		{"Reliable", ProtocolUDPFECReliable, true, false},
		{"Lossy", ProtocolUDPFECLossy, false, true},
		{"Both", ProtocolUDPFECReliable | ProtocolUDPFECLossy, true, true},
		{"None", 0, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &MultitransportRequest{RequestedProtocol: tc.protocol}
			if req.IsReliable() != tc.isReliable {
				t.Errorf("IsReliable: expected %v, got %v", tc.isReliable, req.IsReliable())
			}
			if req.IsLossy() != tc.isLossy {
				t.Errorf("IsLossy: expected %v, got %v", tc.isLossy, req.IsLossy())
			}
		})
	}
}

func TestMultitransportResponse_Serialize(t *testing.T) {
	resp := &MultitransportResponse{
		RequestID: 0x12345678,
		HResult:   HResultSuccess,
	}

	data, err := resp.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	if len(data) != MinResponseSize {
		t.Errorf("Expected %d bytes, got %d", MinResponseSize, len(data))
	}

	// Verify deserialization round-trip
	resp2 := &MultitransportResponse{}
	if err := resp2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if resp2.RequestID != resp.RequestID {
		t.Errorf("RequestID mismatch: %d vs %d", resp2.RequestID, resp.RequestID)
	}
	if resp2.HResult != resp.HResult {
		t.Errorf("HResult mismatch: 0x%X vs 0x%X", resp2.HResult, resp.HResult)
	}
}

func TestMultitransportResponse_Deserialize_TooShort(t *testing.T) {
	resp := &MultitransportResponse{}
	err := resp.Deserialize(make([]byte, 4))
	if err == nil {
		t.Error("Expected error for short data")
	}
}

func TestMultitransportResponse_IsSuccess(t *testing.T) {
	tests := []struct {
		name      string
		hresult   uint32
		isSuccess bool
	}{
		{"Success", HResultSuccess, true},
		{"Abort", HResultAbort, false},
		{"NotFound", HResultNotFound, false},
		{"NoMem", HResultNoMem, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := &MultitransportResponse{HResult: tc.hresult}
			if resp.IsSuccess() != tc.isSuccess {
				t.Errorf("IsSuccess: expected %v, got %v", tc.isSuccess, resp.IsSuccess())
			}
		})
	}
}

func TestNewDeclineResponse(t *testing.T) {
	resp := NewDeclineResponse(42)
	if resp.RequestID != 42 {
		t.Errorf("Expected RequestID 42, got %d", resp.RequestID)
	}
	if resp.HResult != HResultAbort {
		t.Errorf("Expected HResult E_ABORT, got 0x%X", resp.HResult)
	}
}

func TestNewSuccessResponse(t *testing.T) {
	resp := NewSuccessResponse(99)
	if resp.RequestID != 99 {
		t.Errorf("Expected RequestID 99, got %d", resp.RequestID)
	}
	if resp.HResult != HResultSuccess {
		t.Errorf("Expected HResult S_OK, got 0x%X", resp.HResult)
	}
}

func TestTunnelHeader_Serialize(t *testing.T) {
	h := &TunnelHeader{
		Action:        ActionCreateRequest,
		Flags:         0,
		PayloadLength: 100,
	}

	data, err := h.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	if len(data) != TunnelHeaderMinSize {
		t.Errorf("Expected %d bytes, got %d", TunnelHeaderMinSize, len(data))
	}

	// Verify deserialization
	h2 := &TunnelHeader{}
	if err := h2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if h2.Action != h.Action || h2.Flags != h.Flags || h2.PayloadLength != h.PayloadLength {
		t.Error("Header mismatch after round-trip")
	}
	if h2.HeaderLength != TunnelHeaderMinSize {
		t.Errorf("Expected HeaderLength %d, got %d", TunnelHeaderMinSize, h2.HeaderLength)
	}
}

func TestTunnelHeader_ActionFlagsEncoding(t *testing.T) {
	// Test that Action and Flags are correctly encoded in a single byte
	// Per MS-RDPEMT Section 2.2.1.1:
	// - Action is lower 4 bits (nibble)
	// - Flags is upper 4 bits (nibble)
	tests := []struct {
		name   string
		action uint8
		flags  uint8
	}{
		{"CreateRequest", ActionCreateRequest, 0},
		{"CreateResponse", ActionCreateResponse, 0},
		{"Data", ActionData, 0},
		{"DataWithFlags", ActionData, 0x0F}, // Max flags value (though spec says must be 0)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &TunnelHeader{
				Action:        tc.action,
				Flags:         tc.flags,
				PayloadLength: 0,
			}

			data, err := h.Serialize()
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			// Verify byte 0 encoding: (Flags << 4) | Action
			expectedByte0 := (tc.flags << 4) | tc.action
			if data[0] != expectedByte0 {
				t.Errorf("Byte 0: expected 0x%02X, got 0x%02X", expectedByte0, data[0])
			}

			// Verify deserialization
			h2 := &TunnelHeader{}
			if err := h2.Deserialize(data); err != nil {
				t.Fatalf("Deserialize failed: %v", err)
			}

			if h2.Action != tc.action {
				t.Errorf("Action: expected %d, got %d", tc.action, h2.Action)
			}
			if h2.Flags != tc.flags {
				t.Errorf("Flags: expected %d, got %d", tc.flags, h2.Flags)
			}
		})
	}
}

func TestTunnelHeader_WithSubHeaders(t *testing.T) {
	// Test header with SubHeaders (HeaderLength > 4)
	subHeaders := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	h := &TunnelHeader{
		Action:        ActionData,
		Flags:         0,
		PayloadLength: 100,
		SubHeaders:    subHeaders,
	}

	data, err := h.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	expectedLen := TunnelHeaderMinSize + len(subHeaders)
	if len(data) != expectedLen {
		t.Errorf("Expected %d bytes, got %d", expectedLen, len(data))
	}

	// Verify HeaderLength field
	if data[3] != byte(expectedLen) {
		t.Errorf("HeaderLength: expected %d, got %d", expectedLen, data[3])
	}

	// Verify deserialization
	h2 := &TunnelHeader{}
	if err := h2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if !bytes.Equal(h2.SubHeaders, subHeaders) {
		t.Errorf("SubHeaders mismatch: expected %v, got %v", subHeaders, h2.SubHeaders)
	}
}

func TestTunnelHeader_Deserialize_TooShort(t *testing.T) {
	h := &TunnelHeader{}
	err := h.Deserialize(make([]byte, 2))
	if err == nil {
		t.Error("Expected error for short data")
	}
}

func TestTunnelCreateRequest_RoundTrip(t *testing.T) {
	req := &TunnelCreateRequest{
		RequestID: 0xDEADBEEF,
		Reserved:  0,
	}
	copy(req.SecurityCookie[:], []byte("0123456789ABCDEF"))

	data, err := req.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	if len(data) != req.Size() {
		t.Errorf("Expected %d bytes, got %d", req.Size(), len(data))
	}

	req2 := &TunnelCreateRequest{}
	if err := req2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if req2.RequestID != req.RequestID {
		t.Error("RequestID mismatch")
	}
	if !bytes.Equal(req2.SecurityCookie[:], req.SecurityCookie[:]) {
		t.Error("SecurityCookie mismatch")
	}
}

func TestTunnelCreateRequest_Deserialize_TooShort(t *testing.T) {
	req := &TunnelCreateRequest{}
	err := req.Deserialize(make([]byte, 10))
	if err == nil {
		t.Error("Expected error for short data")
	}
}

func TestTunnelCreateResponse_RoundTrip(t *testing.T) {
	resp := &TunnelCreateResponse{HResult: HResultSuccess}

	data, err := resp.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	if len(data) != resp.Size() {
		t.Errorf("Expected %d bytes, got %d", resp.Size(), len(data))
	}

	resp2 := &TunnelCreateResponse{}
	if err := resp2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if resp2.HResult != resp.HResult {
		t.Error("HResult mismatch")
	}
}

func TestTunnelCreateResponse_Deserialize_TooShort(t *testing.T) {
	resp := &TunnelCreateResponse{}
	err := resp.Deserialize(make([]byte, 2))
	if err == nil {
		t.Error("Expected error for short data")
	}
}

func TestTunnelDataPDU_RoundTrip(t *testing.T) {
	pdu := &TunnelDataPDU{
		Data: []byte("Hello, RDP over UDP!"),
	}

	data, err := pdu.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	pdu2 := &TunnelDataPDU{}
	if err := pdu2.Deserialize(data); err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if pdu2.Header.Action != ActionData {
		t.Errorf("Expected Action %d, got %d", ActionData, pdu2.Header.Action)
	}
	if !bytes.Equal(pdu2.Data, pdu.Data) {
		t.Error("Data mismatch")
	}
}

func TestTunnelDataPDU_Deserialize_WrongAction(t *testing.T) {
	header := &TunnelHeader{
		Action:        ActionCreateRequest,
		PayloadLength: 10,
	}
	headerBytes, _ := header.Serialize()
	data := append(headerBytes, make([]byte, 10)...)

	pdu := &TunnelDataPDU{}
	err := pdu.Deserialize(data)
	if err == nil {
		t.Error("Expected error for wrong action type")
	}
}

func TestTunnelDataPDU_Deserialize_TruncatedPayload(t *testing.T) {
	header := &TunnelHeader{
		Action:        ActionData,
		PayloadLength: 100,
	}
	headerBytes, _ := header.Serialize()
	data := append(headerBytes, make([]byte, 10)...) // Only 10 bytes, but header says 100

	pdu := &TunnelDataPDU{}
	err := pdu.Deserialize(data)
	if err == nil {
		t.Error("Expected error for truncated payload")
	}
}

func TestParseTunnelPDU(t *testing.T) {
	tests := []struct {
		name   string
		action uint8
	}{
		{"CreateRequest", ActionCreateRequest},
		{"CreateResponse", ActionCreateResponse},
		{"Data", ActionData},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := []byte("test payload")
			header := &TunnelHeader{
				Action:        tc.action,
				PayloadLength: uint16(len(payload)),
			}
			headerBytes, _ := header.Serialize()
			data := append(headerBytes, payload...)

			action, parsedPayload, err := ParseTunnelPDU(data)
			if err != nil {
				t.Fatalf("ParseTunnelPDU failed: %v", err)
			}
			if action != tc.action {
				t.Errorf("Action mismatch: expected %d, got %d", tc.action, action)
			}
			if !bytes.Equal(parsedPayload, payload) {
				t.Error("Payload mismatch")
			}
		})
	}
}

func TestParseTunnelPDU_TooShort(t *testing.T) {
	_, _, err := ParseTunnelPDU(make([]byte, 2))
	if err == nil {
		t.Error("Expected error for short data")
	}
}

func TestParseTunnelPDU_TruncatedPayload(t *testing.T) {
	header := &TunnelHeader{
		Action:        ActionData,
		PayloadLength: 100,
	}
	headerBytes, _ := header.Serialize()
	data := append(headerBytes, make([]byte, 10)...)

	_, _, err := ParseTunnelPDU(data)
	if err == nil {
		t.Error("Expected error for truncated payload")
	}
}

func TestHResultString(t *testing.T) {
	tests := []struct {
		hr       uint32
		expected string
	}{
		{HResultSuccess, "S_OK"},
		{HResultNoMem, "E_OUTOFMEMORY"},
		{HResultNotFound, "E_NOTFOUND"},
		{HResultAbort, "E_ABORT"},
		{0x12345678, "0x12345678"},
	}

	for _, tc := range tests {
		result := HResultString(tc.hr)
		if result != tc.expected {
			t.Errorf("HResultString(0x%X): expected %q, got %q", tc.hr, tc.expected, result)
		}
	}
}

func TestProtocolString(t *testing.T) {
	tests := []struct {
		proto    uint16
		expected string
	}{
		{0, "None"},
		{ProtocolUDPFECReliable, "[UDP-FEC-Reliable]"},
		{ProtocolUDPFECLossy, "[UDP-FEC-Lossy]"},
		{ProtocolUDPFECReliable | ProtocolUDPFECLossy, "[UDP-FEC-Reliable UDP-FEC-Lossy]"},
	}

	for _, tc := range tests {
		result := ProtocolString(tc.proto)
		if result != tc.expected {
			t.Errorf("ProtocolString(0x%X): expected %q, got %q", tc.proto, tc.expected, result)
		}
	}
}

func TestMultitransportRequest_FreeRDPCompatibility(t *testing.T) {
	// Test data that would be received from a real RDP server
	// This matches the format in FreeRDP's multitransport_recv_request()
	data := make([]byte, 24)
	// RequestID = 1
	data[0] = 0x01
	data[1] = 0x00
	data[2] = 0x00
	data[3] = 0x00
	// RequestedProtocol = UDPFECR (0x0001)
	data[4] = 0x01
	data[5] = 0x00
	// Reserved = 0
	data[6] = 0x00
	data[7] = 0x00
	// SecurityCookie (16 bytes)
	for i := 0; i < 16; i++ {
		data[8+i] = byte(i)
	}

	req := &MultitransportRequest{}
	if err := req.Deserialize(data); err != nil {
		t.Fatalf("Failed to parse FreeRDP-compatible data: %v", err)
	}

	if req.RequestID != 1 {
		t.Errorf("RequestID: expected 1, got %d", req.RequestID)
	}
	if req.RequestedProtocol != ProtocolUDPFECReliable {
		t.Errorf("Protocol: expected reliable UDP, got 0x%04X", req.RequestedProtocol)
	}
	if !req.IsReliable() {
		t.Error("Expected IsReliable() to return true")
	}
}
