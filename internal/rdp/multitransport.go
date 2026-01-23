package rdp

import (
	"crypto/rand"
	"errors"
	"log"
	"sync"

	"github.com/rcarmo/rdp-html5/internal/protocol/rdpemt"
)

// MultitransportHandler manages the multitransport negotiation for UDP transport.
// This implements the client side of MS-RDPEMT for handling server requests to
// establish UDP transport channels.
type MultitransportHandler struct {
	mu sync.Mutex

	// Connection for sending responses
	sendFunc func(data []byte) error

	// State tracking
	pendingRequests map[uint32]*rdpemt.MultitransportRequest

	// Configuration
	udpEnabled bool // Whether to accept UDP transport requests

	// Callbacks (optional)
	onUDPReady func(requestID uint32, cookie [16]byte, reliable bool)
}

// NewMultitransportHandler creates a new multitransport handler.
func NewMultitransportHandler(sendFunc func(data []byte) error) *MultitransportHandler {
	return &MultitransportHandler{
		sendFunc:        sendFunc,
		pendingRequests: make(map[uint32]*rdpemt.MultitransportRequest),
		udpEnabled:      false, // Disabled by default
	}
}

// EnableUDP enables UDP transport handling.
func (h *MultitransportHandler) EnableUDP(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.udpEnabled = enabled
}

// SetUDPReadyCallback sets a callback for when UDP transport is ready.
func (h *MultitransportHandler) SetUDPReadyCallback(cb func(requestID uint32, cookie [16]byte, reliable bool)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onUDPReady = cb
}

// HandleRequest processes a server multitransport request.
// If UDP is disabled, it automatically responds with E_ABORT.
// If UDP is enabled, it stores the request for later processing and notifies via callback.
func (h *MultitransportHandler) HandleRequest(data []byte) error {
	var req rdpemt.MultitransportRequest
	if err := req.Deserialize(data); err != nil {
		return err
	}

	h.mu.Lock()
	enabled := h.udpEnabled
	callback := h.onUDPReady

	if !enabled {
		h.mu.Unlock()
		// Decline the request (like FreeRDP's multitransport_no_udp)
		return h.sendDecline(req.RequestID)
	}

	// Store the request for later processing
	h.pendingRequests[req.RequestID] = &req
	h.mu.Unlock()

	log.Printf("Multitransport request received: ID=%d, Protocol=%s",
		req.RequestID, rdpemt.ProtocolString(req.RequestedProtocol))

	// Notify callback if set
	if callback != nil {
		callback(req.RequestID, req.SecurityCookie, req.IsReliable())
	}

	return nil
}

// sendDecline sends a decline response for a multitransport request.
func (h *MultitransportHandler) sendDecline(requestID uint32) error {
	resp := rdpemt.NewDeclineResponse(requestID)
	data, err := resp.Serialize()
	if err != nil {
		return err
	}

	log.Printf("Declining multitransport request ID=%d (UDP disabled)", requestID)
	return h.sendFunc(data)
}

// AcceptRequest accepts a pending multitransport request.
// This should be called after establishing the UDP connection.
func (h *MultitransportHandler) AcceptRequest(requestID uint32) error {
	h.mu.Lock()
	_, exists := h.pendingRequests[requestID]
	if exists {
		delete(h.pendingRequests, requestID)
	}
	h.mu.Unlock()

	if !exists {
		return errors.New("no pending request with that ID")
	}

	resp := rdpemt.NewSuccessResponse(requestID)
	data, err := resp.Serialize()
	if err != nil {
		return err
	}

	log.Printf("Accepting multitransport request ID=%d", requestID)
	return h.sendFunc(data)
}

// DeclineRequest explicitly declines a pending multitransport request.
func (h *MultitransportHandler) DeclineRequest(requestID uint32) error {
	h.mu.Lock()
	_, exists := h.pendingRequests[requestID]
	if exists {
		delete(h.pendingRequests, requestID)
	}
	h.mu.Unlock()

	if !exists {
		return errors.New("no pending request with that ID")
	}

	return h.sendDecline(requestID)
}

// GetPendingRequest returns a pending request by ID.
func (h *MultitransportHandler) GetPendingRequest(requestID uint32) *rdpemt.MultitransportRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.pendingRequests[requestID]
}

// ClearPendingRequests removes all pending requests.
func (h *MultitransportHandler) ClearPendingRequests() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pendingRequests = make(map[uint32]*rdpemt.MultitransportRequest)
}

// GenerateCookie generates a random security cookie for tunnel binding.
func GenerateCookie() ([16]byte, error) {
	var cookie [16]byte
	_, err := rand.Read(cookie[:])
	return cookie, err
}

// GenerateCookieHash generates a hash of the security cookie.
// The hash algorithm is not specified in MS-RDPEMT, but FreeRDP uses
// the cookie directly as the hash in most cases.
func GenerateCookieHash(cookie [16]byte) [32]byte {
	var hash [32]byte
	// For compatibility, just copy the cookie twice
	// Real implementations may use SHA-256 or similar
	copy(hash[0:16], cookie[:])
	copy(hash[16:32], cookie[:])
	return hash
}
