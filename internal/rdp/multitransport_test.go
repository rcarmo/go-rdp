package rdp

import (
	"bytes"
	"testing"

	"github.com/rcarmo/rdp-html5/internal/protocol/rdpemt"
)

func TestNewMultitransportHandler(t *testing.T) {
	handler := NewMultitransportHandler(func(data []byte) error {
		return nil
	})

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	if handler.udpEnabled {
		t.Error("UDP should be disabled by default")
	}
}

func TestMultitransportHandler_EnableUDP(t *testing.T) {
	handler := NewMultitransportHandler(func(data []byte) error { return nil })

	handler.EnableUDP(true)
	if !handler.udpEnabled {
		t.Error("UDP should be enabled")
	}

	handler.EnableUDP(false)
	if handler.udpEnabled {
		t.Error("UDP should be disabled")
	}
}

func TestMultitransportHandler_HandleRequest_UDPDisabled(t *testing.T) {
	var sentData []byte
	handler := NewMultitransportHandler(func(data []byte) error {
		sentData = data
		return nil
	})

	// Create a request
	req := &rdpemt.MultitransportRequest{
		RequestID:         42,
		RequestedProtocol: rdpemt.ProtocolUDPFECReliable,
	}
	copy(req.SecurityCookie[:], []byte("0123456789ABCDEF"))
	reqData, _ := req.Serialize()

	// Handle the request (UDP disabled by default)
	if err := handler.HandleRequest(reqData); err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	// Should have sent a decline response
	if sentData == nil {
		t.Fatal("Expected response to be sent")
	}

	// Verify it's a decline
	var resp rdpemt.MultitransportResponse
	if err := resp.Deserialize(sentData); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.RequestID != 42 {
		t.Errorf("Expected RequestID 42, got %d", resp.RequestID)
	}
	if resp.HResult != rdpemt.HResultAbort {
		t.Errorf("Expected E_ABORT, got 0x%X", resp.HResult)
	}
}

func TestMultitransportHandler_HandleRequest_UDPEnabled(t *testing.T) {
	var sentData []byte
	callbackCalled := false
	var callbackRequestID uint32
	var callbackCookie [16]byte
	var callbackReliable bool

	handler := NewMultitransportHandler(func(data []byte) error {
		sentData = data
		return nil
	})

	handler.EnableUDP(true)
	handler.SetUDPReadyCallback(func(reqID uint32, cookie [16]byte, reliable bool) {
		callbackCalled = true
		callbackRequestID = reqID
		callbackCookie = cookie
		callbackReliable = reliable
	})

	// Create a request
	req := &rdpemt.MultitransportRequest{
		RequestID:         99,
		RequestedProtocol: rdpemt.ProtocolUDPFECReliable,
	}
	copy(req.SecurityCookie[:], []byte("SecretCookie1234"))
	reqData, _ := req.Serialize()

	// Handle the request
	if err := handler.HandleRequest(reqData); err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	// Should NOT have sent a response (request is pending)
	if sentData != nil {
		t.Error("Should not send response immediately when UDP is enabled")
	}

	// Callback should have been called
	if !callbackCalled {
		t.Error("Expected callback to be called")
	}
	if callbackRequestID != 99 {
		t.Errorf("Expected callback RequestID 99, got %d", callbackRequestID)
	}
	if !callbackReliable {
		t.Error("Expected reliable=true in callback")
	}
	if !bytes.Equal(callbackCookie[:], []byte("SecretCookie1234")) {
		t.Error("Cookie mismatch in callback")
	}

	// Verify request is pending
	pending := handler.GetPendingRequest(99)
	if pending == nil {
		t.Error("Expected pending request")
	}
}

func TestMultitransportHandler_AcceptRequest(t *testing.T) {
	var sentData []byte
	handler := NewMultitransportHandler(func(data []byte) error {
		sentData = data
		return nil
	})

	handler.EnableUDP(true)

	// Create and handle a request
	req := &rdpemt.MultitransportRequest{
		RequestID:         100,
		RequestedProtocol: rdpemt.ProtocolUDPFECReliable,
	}
	reqData, _ := req.Serialize()
	handler.HandleRequest(reqData)

	// Accept the request
	if err := handler.AcceptRequest(100); err != nil {
		t.Fatalf("AcceptRequest failed: %v", err)
	}

	// Verify response
	if sentData == nil {
		t.Fatal("Expected response to be sent")
	}

	var resp rdpemt.MultitransportResponse
	if err := resp.Deserialize(sentData); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.RequestID != 100 {
		t.Errorf("Expected RequestID 100, got %d", resp.RequestID)
	}
	if resp.HResult != rdpemt.HResultSuccess {
		t.Errorf("Expected S_OK, got 0x%X", resp.HResult)
	}

	// Request should no longer be pending
	pending := handler.GetPendingRequest(100)
	if pending != nil {
		t.Error("Request should not be pending after accept")
	}
}

func TestMultitransportHandler_AcceptRequest_NotPending(t *testing.T) {
	handler := NewMultitransportHandler(func(data []byte) error { return nil })

	err := handler.AcceptRequest(999)
	if err == nil {
		t.Error("Expected error for non-pending request")
	}
}

func TestMultitransportHandler_DeclineRequest(t *testing.T) {
	var sentData []byte
	handler := NewMultitransportHandler(func(data []byte) error {
		sentData = data
		return nil
	})

	handler.EnableUDP(true)

	// Create and handle a request
	req := &rdpemt.MultitransportRequest{
		RequestID:         200,
		RequestedProtocol: rdpemt.ProtocolUDPFECLossy,
	}
	reqData, _ := req.Serialize()
	handler.HandleRequest(reqData)

	// Decline the request
	if err := handler.DeclineRequest(200); err != nil {
		t.Fatalf("DeclineRequest failed: %v", err)
	}

	// Verify response
	var resp rdpemt.MultitransportResponse
	if err := resp.Deserialize(sentData); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.HResult != rdpemt.HResultAbort {
		t.Errorf("Expected E_ABORT, got 0x%X", resp.HResult)
	}

	// Request should no longer be pending
	pending := handler.GetPendingRequest(200)
	if pending != nil {
		t.Error("Request should not be pending after decline")
	}
}

func TestMultitransportHandler_DeclineRequest_NotPending(t *testing.T) {
	handler := NewMultitransportHandler(func(data []byte) error { return nil })

	err := handler.DeclineRequest(999)
	if err == nil {
		t.Error("Expected error for non-pending request")
	}
}

func TestMultitransportHandler_ClearPendingRequests(t *testing.T) {
	handler := NewMultitransportHandler(func(data []byte) error { return nil })
	handler.EnableUDP(true)

	// Create some pending requests
	for i := uint32(1); i <= 5; i++ {
		req := &rdpemt.MultitransportRequest{RequestID: i}
		reqData, _ := req.Serialize()
		handler.HandleRequest(reqData)
	}

	// Verify they're pending
	for i := uint32(1); i <= 5; i++ {
		if handler.GetPendingRequest(i) == nil {
			t.Errorf("Request %d should be pending", i)
		}
	}

	// Clear all
	handler.ClearPendingRequests()

	// Verify they're gone
	for i := uint32(1); i <= 5; i++ {
		if handler.GetPendingRequest(i) != nil {
			t.Errorf("Request %d should not be pending after clear", i)
		}
	}
}

func TestMultitransportHandler_HandleRequest_InvalidData(t *testing.T) {
	handler := NewMultitransportHandler(func(data []byte) error { return nil })

	// Try to handle invalid data
	err := handler.HandleRequest([]byte{0x01, 0x02}) // Too short
	if err == nil {
		t.Error("Expected error for invalid data")
	}
}

func TestGenerateCookie(t *testing.T) {
	cookie1, err := GenerateCookie()
	if err != nil {
		t.Fatalf("GenerateCookie failed: %v", err)
	}

	cookie2, err := GenerateCookie()
	if err != nil {
		t.Fatalf("GenerateCookie failed: %v", err)
	}

	// Cookies should be different (random)
	if bytes.Equal(cookie1[:], cookie2[:]) {
		t.Error("Expected different cookies from consecutive calls")
	}

	// Cookie should not be all zeros
	allZeros := true
	for _, b := range cookie1 {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		t.Error("Cookie should not be all zeros")
	}
}

func TestGenerateCookieHash(t *testing.T) {
	cookie := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	hash := GenerateCookieHash(cookie)

	// Hash should be 32 bytes
	if len(hash) != 32 {
		t.Errorf("Expected 32-byte hash, got %d", len(hash))
	}

	// First 16 bytes should match cookie
	if !bytes.Equal(hash[0:16], cookie[:]) {
		t.Error("First half of hash should match cookie")
	}
}
