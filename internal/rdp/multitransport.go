package rdp

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rcarmo/go-rdp/internal/protocol/rdpemt"
	"github.com/rcarmo/go-rdp/internal/transport/udp"
)

// MultitransportHandler manages the multitransport negotiation for UDP transport.
// This implements the client side of MS-RDPBCGR Section 3.2.5.15.1 for handling
// server requests to establish UDP transport channels.
//
// Reference: MS-RDPBCGR Section 3.2.5.15.1 - Processing the Initiate Multitransport Request PDU
// Pseudo-code from spec:
//
//	1. Decode the PDU (requestId, requestedProtocol, securityCookie)
//	2. Attempt to create sideband channel using requested protocol
//	3. If successful, secure channel with TLS (reliable) or DTLS (lossy)
//	4. Send Tunnel Create Request PDU over secure channel
//	5. If failed or Soft-Sync negotiated, send Initiate Multitransport Response PDU
type MultitransportHandler struct {
	mu sync.Mutex

	// Connection for sending responses
	sendFunc func(data []byte) error

	// State tracking
	pendingRequests map[uint32]*rdpemt.MultitransportRequest

	// Configuration
	udpEnabled bool // Whether to accept UDP transport requests

	// Tunnel manager for UDP connections
	tunnelMgr *udp.TunnelManager

	// Server address (extracted from RDP connection)
	serverHost string
	serverPort int

	// Soft-Sync support (TCP to UDP channel migration)
	softSyncSupported bool

	// Callbacks (optional)
	onUDPReady   func(requestID uint32, cookie [16]byte, reliable bool)
	onTunnelData func(requestID uint32, data []byte)
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
// Per MS-RDPBCGR: If unable to initiate sideband channel, SHOULD send decline response.
func (h *MultitransportHandler) EnableUDP(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.udpEnabled = enabled
}

// SetServerAddress sets the server address for UDP connections.
// This should be called with the same host:port as the TCP connection.
func (h *MultitransportHandler) SetServerAddress(host string, port int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.serverHost = host
	h.serverPort = port

	// Initialize tunnel manager if UDP is enabled
	if h.udpEnabled && h.tunnelMgr == nil {
		h.initTunnelManager()
	}
}

// SetSoftSyncSupported sets whether Soft-Sync is supported.
// Per MS-RDPBCGR: If Soft-Sync supported, MUST send response regardless of success/failure.
func (h *MultitransportHandler) SetSoftSyncSupported(supported bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.softSyncSupported = supported
}

// initTunnelManager initializes the tunnel manager
func (h *MultitransportHandler) initTunnelManager() {
	if h.serverHost == "" {
		return
	}

	serverAddr := net.JoinHostPort(h.serverHost, fmt.Sprintf("%d", h.serverPort))
	var err error
	h.tunnelMgr, err = udp.NewTunnelManager(&udp.TunnelManagerConfig{
		ServerAddr:      serverAddr,
		Enabled:         h.udpEnabled,
		ConnectTimeout:  10 * time.Second,
		ProtocolVersion: 0x0002, // Version 2
	})

	if err != nil {
		log.Printf("Failed to create tunnel manager: %v", err)
		return
	}

	// Set up callbacks
	h.tunnelMgr.SetCallbacks(
		h.onTunnelEstablished,
		h.onTunnelClosed,
		h.onTunnelDataReceived,
	)
}

// SetUDPReadyCallback sets a callback for when UDP transport is ready.
func (h *MultitransportHandler) SetUDPReadyCallback(cb func(requestID uint32, cookie [16]byte, reliable bool)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onUDPReady = cb
}

// SetTunnelDataCallback sets a callback for data received over UDP tunnel.
func (h *MultitransportHandler) SetTunnelDataCallback(cb func(requestID uint32, data []byte)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onTunnelData = cb
}

// HandleRequest processes a server multitransport request.
// Per MS-RDPBCGR Section 3.2.5.15.1:
//
//	1. Decode PDU
//	2. Attempt to create sideband channel
//	3. Secure with TLS/DTLS
//	4. Send Tunnel Create Request
//	5. Send response if failed or Soft-Sync
func (h *MultitransportHandler) HandleRequest(data []byte) error {
	var req rdpemt.MultitransportRequest
	if err := req.Deserialize(data); err != nil {
		return err
	}

	h.mu.Lock()
	enabled := h.udpEnabled
	callback := h.onUDPReady
	softSync := h.softSyncSupported
	tunnelMgr := h.tunnelMgr

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

	// Attempt to establish UDP tunnel
	if tunnelMgr != nil {
		err := tunnelMgr.HandleMultitransportRequest(&req)
		if err != nil {
			log.Printf("Failed to initiate UDP tunnel: %v", err)
			h.mu.Lock()
			delete(h.pendingRequests, req.RequestID)
			h.mu.Unlock()
			// Per spec: If unable to initiate, SHOULD send decline
			if softSync {
				// Soft-Sync: MUST send response regardless
				return h.sendDecline(req.RequestID)
			}
			return h.sendDecline(req.RequestID)
		}
		// Tunnel establishment continues asynchronously
		// Response will be sent in onTunnelEstablished callback
	} else {
		// No tunnel manager - notify callback for manual handling
		if callback != nil {
			callback(req.RequestID, req.SecurityCookie, req.IsReliable())
		}
	}

	return nil
}

// onTunnelEstablished is called when a UDP tunnel is successfully established
func (h *MultitransportHandler) onTunnelEstablished(tunnel *udp.Tunnel) {
	h.mu.Lock()
	req := h.pendingRequests[tunnel.RequestID]
	callback := h.onUDPReady
	softSync := h.softSyncSupported
	h.mu.Unlock()

	if req == nil {
		log.Printf("Tunnel established for unknown request ID: %d", tunnel.RequestID)
		return
	}

	h.mu.Lock()
	delete(h.pendingRequests, tunnel.RequestID)
	h.mu.Unlock()

	log.Printf("UDP tunnel %d established successfully", tunnel.RequestID)

	// Per MS-RDPBCGR: If Soft-Sync supported, MUST send success response
	if softSync {
		h.AcceptRequest(tunnel.RequestID)
	}

	// Notify callback
	if callback != nil {
		callback(tunnel.RequestID, req.SecurityCookie, req.IsReliable())
	}
}

// onTunnelClosed is called when a UDP tunnel is closed
func (h *MultitransportHandler) onTunnelClosed(requestID uint32, err error) {
	h.mu.Lock()
	req := h.pendingRequests[requestID]
	softSync := h.softSyncSupported
	delete(h.pendingRequests, requestID)
	h.mu.Unlock()

	if err != nil {
		log.Printf("UDP tunnel %d failed: %v", requestID, err)
		// Per spec: If sideband creation fails and Soft-Sync, send decline
		if req != nil && softSync {
			h.sendDecline(requestID)
		}
	} else {
		log.Printf("UDP tunnel %d closed", requestID)
	}
}

// onTunnelDataReceived is called when data is received on a UDP tunnel
func (h *MultitransportHandler) onTunnelDataReceived(requestID uint32, data []byte) {
	h.mu.Lock()
	callback := h.onTunnelData
	h.mu.Unlock()

	if callback != nil {
		callback(requestID, data)
	}
}

// sendDecline sends a decline response for a multitransport request.
func (h *MultitransportHandler) sendDecline(requestID uint32) error {
	resp := rdpemt.NewDeclineResponse(requestID)
	data, err := resp.Serialize()
	if err != nil {
		return err
	}

	log.Printf("Declining multitransport request ID=%d", requestID)
	return h.sendFunc(data)
}

// AcceptRequest accepts a pending multitransport request.
// Per MS-RDPBCGR Section 3.2.5.15.2: Send success response after UDP established.
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

// SendOverUDP sends data over a specific UDP tunnel.
func (h *MultitransportHandler) SendOverUDP(requestID uint32, data []byte) error {
	h.mu.Lock()
	tunnelMgr := h.tunnelMgr
	h.mu.Unlock()

	if tunnelMgr == nil {
		return errors.New("UDP tunnels not available")
	}

	return tunnelMgr.SendData(requestID, data)
}

// GetTunnelState returns the state of a UDP tunnel.
func (h *MultitransportHandler) GetTunnelState(requestID uint32) string {
	h.mu.Lock()
	tunnelMgr := h.tunnelMgr
	h.mu.Unlock()

	if tunnelMgr == nil {
		return "NO_TUNNEL_MANAGER"
	}

	tunnel := tunnelMgr.GetTunnel(requestID)
	if tunnel == nil {
		return "NOT_FOUND"
	}

	return tunnel.State().String()
}

// Close closes all UDP tunnels and cleans up resources.
func (h *MultitransportHandler) Close() {
	h.mu.Lock()
	tunnelMgr := h.tunnelMgr
	h.tunnelMgr = nil
	h.mu.Unlock()

	if tunnelMgr != nil {
		tunnelMgr.CloseAll()
	}
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

// ExtractHostPort extracts host and port from an address string.
// Helper for setting server address from RDP connection string.
func ExtractHostPort(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		// No port specified, assume default RDP port
		return strings.TrimSuffix(addr, ":"), 3389
	}
	port := 3389
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 && p < 65536 {
			port = p
		}
	}
	return host, port
}
