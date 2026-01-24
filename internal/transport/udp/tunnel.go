// Package udp implements the tunnel manager for MS-RDPEMT multitransport.
// This file manages the lifecycle of UDP tunnels in coordination with
// the main RDP connection.
//
// Reference: MS-RDPEMT Section 3 - Protocol Details
// Reference: MS-RDPEMT_ServerTestDesignSpecification.md
package udp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/rcarmo/go-rdp/internal/protocol/rdpemt"
)

// TunnelState represents the state of a UDP tunnel
type TunnelState int

const (
	TunnelStateInit TunnelState = iota
	TunnelStateConnecting
	TunnelStateSecuring
	TunnelStateTunnelCreate
	TunnelStateEstablished
	TunnelStateClosed
	TunnelStateFailed
)

func (s TunnelState) String() string {
	switch s {
	case TunnelStateInit:
		return "INIT"
	case TunnelStateConnecting:
		return "CONNECTING"
	case TunnelStateSecuring:
		return "SECURING"
	case TunnelStateTunnelCreate:
		return "TUNNEL_CREATE"
	case TunnelStateEstablished:
		return "ESTABLISHED"
	case TunnelStateClosed:
		return "CLOSED"
	case TunnelStateFailed:
		return "FAILED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}

// Tunnel represents an active UDP tunnel
type Tunnel struct {
	mu sync.RWMutex

	// Tunnel identification
	RequestID      uint32
	SecurityCookie [16]byte
	Reliable       bool

	// Connection
	secureConn *SecureConnection

	// State
	state     TunnelState
	lastError error

	// Data channel
	dataChan chan []byte
	closeCh  chan struct{}
}

// TunnelManager manages UDP tunnels for an RDP session
type TunnelManager struct {
	mu sync.RWMutex

	// Remote server address (same as TCP connection)
	serverAddr *net.UDPAddr

	// Active tunnels by RequestID
	tunnels map[uint32]*Tunnel

	// Configuration
	enabled         bool
	connectTimeout  time.Duration
	protocolVersion uint16

	// Callbacks
	onTunnelReady    func(tunnel *Tunnel)
	onTunnelClosed   func(requestID uint32, err error)
	onChannelData    func(requestID uint32, data []byte)
}

// TunnelManagerConfig holds tunnel manager configuration
type TunnelManagerConfig struct {
	// ServerAddr is the RDP server address (host:port)
	ServerAddr string

	// Enabled controls whether UDP tunnels are allowed
	Enabled bool

	// ConnectTimeout is the timeout for tunnel establishment
	ConnectTimeout time.Duration

	// ProtocolVersion is the RDPEUDP version to negotiate
	ProtocolVersion uint16
}

// NewTunnelManager creates a new tunnel manager
func NewTunnelManager(config *TunnelManagerConfig) (*TunnelManager, error) {
	if config == nil {
		config = &TunnelManagerConfig{
			ConnectTimeout:  10 * time.Second,
			ProtocolVersion: 0x0002, // Version 2
		}
	}

	tm := &TunnelManager{
		tunnels:         make(map[uint32]*Tunnel),
		enabled:         config.Enabled,
		connectTimeout:  config.ConnectTimeout,
		protocolVersion: config.ProtocolVersion,
	}

	// Parse server address if provided
	if config.ServerAddr != "" {
		addr, err := net.ResolveUDPAddr("udp", config.ServerAddr)
		if err != nil {
			return nil, fmt.Errorf("resolve UDP address: %w", err)
		}
		tm.serverAddr = addr
	}

	return tm, nil
}

// SetServerAddr sets the server address for UDP connections
func (tm *TunnelManager) SetServerAddr(host string, port int) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	tm.mu.Lock()
	tm.serverAddr = addr
	tm.mu.Unlock()
	return nil
}

// SetEnabled enables or disables UDP tunnel support
func (tm *TunnelManager) SetEnabled(enabled bool) {
	tm.mu.Lock()
	tm.enabled = enabled
	tm.mu.Unlock()
}

// IsEnabled returns whether UDP tunnels are enabled
func (tm *TunnelManager) IsEnabled() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.enabled
}

// SetCallbacks sets the tunnel event callbacks
func (tm *TunnelManager) SetCallbacks(
	onReady func(*Tunnel),
	onClosed func(uint32, error),
	onData func(uint32, []byte),
) {
	tm.mu.Lock()
	tm.onTunnelReady = onReady
	tm.onTunnelClosed = onClosed
	tm.onChannelData = onData
	tm.mu.Unlock()
}

// HandleMultitransportRequest processes a server request for UDP transport
// Per MS-RDPEMT Section 3.1.5
func (tm *TunnelManager) HandleMultitransportRequest(req *rdpemt.MultitransportRequest) error {
	tm.mu.Lock()
	if !tm.enabled {
		tm.mu.Unlock()
		log.Printf("UDP tunnel disabled, declining request %d", req.RequestID)
		return nil // Will be declined by caller
	}

	if tm.serverAddr == nil {
		tm.mu.Unlock()
		return errors.New("server address not set")
	}

	serverAddr := tm.serverAddr
	version := tm.protocolVersion
	timeout := tm.connectTimeout
	tm.mu.Unlock()

	// Create tunnel
	tunnel := &Tunnel{
		RequestID:      req.RequestID,
		SecurityCookie: req.SecurityCookie,
		Reliable:       req.IsReliable(),
		state:          TunnelStateInit,
		dataChan:       make(chan []byte, 256),
		closeCh:        make(chan struct{}),
	}

	// Store tunnel
	tm.mu.Lock()
	tm.tunnels[req.RequestID] = tunnel
	tm.mu.Unlock()

	// Start connection in background
	go tm.establishTunnel(tunnel, serverAddr, version, timeout)

	return nil
}

// establishTunnel connects and establishes a UDP tunnel
func (tm *TunnelManager) establishTunnel(
	tunnel *Tunnel,
	serverAddr *net.UDPAddr,
	version uint16,
	timeout time.Duration,
) {
	tunnel.mu.Lock()
	tunnel.state = TunnelStateConnecting
	tunnel.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create secure connection config
	secureConfig := &SecureConfig{
		UDPConfig: &Config{
			RemoteAddr:        serverAddr,
			MTU:               DefaultMTU,
			ReceiveWindowSize: DefaultReceiveWindowSize,
			Reliable:          tunnel.Reliable,
			ProtocolVersion:   version,
		},
		Reliable:       tunnel.Reliable,
		RequestID:      tunnel.RequestID,
		SecurityCookie: tunnel.SecurityCookie,
	}

	// Create secure connection
	secureConn, err := NewSecureConnection(secureConfig)
	if err != nil {
		tm.handleTunnelError(tunnel, fmt.Errorf("create secure connection: %w", err))
		return
	}

	// Connect (RDPEUDP + TLS/DTLS + Tunnel Create)
	if err := secureConn.Connect(ctx); err != nil {
		tm.handleTunnelError(tunnel, fmt.Errorf("connect: %w", err))
		return
	}

	tunnel.mu.Lock()
	tunnel.secureConn = secureConn
	tunnel.state = TunnelStateEstablished
	tunnel.mu.Unlock()

	log.Printf("UDP tunnel %d established (reliable=%v)", tunnel.RequestID, tunnel.Reliable)

	// Notify callback
	tm.mu.RLock()
	onReady := tm.onTunnelReady
	tm.mu.RUnlock()

	if onReady != nil {
		onReady(tunnel)
	}

	// Start receive loop
	go tm.tunnelReceiveLoop(tunnel)
}

// handleTunnelError handles tunnel establishment errors
func (tm *TunnelManager) handleTunnelError(tunnel *Tunnel, err error) {
	tunnel.mu.Lock()
	tunnel.state = TunnelStateFailed
	tunnel.lastError = err
	tunnel.mu.Unlock()

	log.Printf("UDP tunnel %d failed: %v", tunnel.RequestID, err)

	// Notify callback
	tm.mu.RLock()
	onClosed := tm.onTunnelClosed
	tm.mu.RUnlock()

	if onClosed != nil {
		onClosed(tunnel.RequestID, err)
	}

	// Remove from active tunnels
	tm.mu.Lock()
	delete(tm.tunnels, tunnel.RequestID)
	tm.mu.Unlock()
}

// tunnelReceiveLoop reads data from the tunnel
func (tm *TunnelManager) tunnelReceiveLoop(tunnel *Tunnel) {
	buf := make([]byte, 8192)

	for {
		select {
		case <-tunnel.closeCh:
			return
		default:
		}

		tunnel.mu.RLock()
		conn := tunnel.secureConn
		tunnel.mu.RUnlock()

		if conn == nil {
			return
		}

		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				tm.handleTunnelError(tunnel, err)
			} else {
				tm.closeTunnel(tunnel, nil)
			}
			return
		}

		if n > 0 {
			// Parse TunnelDataPDU
			action, payload, err := rdpemt.ParseTunnelPDU(buf[:n])
			if err != nil {
				log.Printf("Tunnel %d: parse error: %v", tunnel.RequestID, err)
				continue
			}

			if action == rdpemt.ActionData {
				// Deliver to callback
				tm.mu.RLock()
				onData := tm.onChannelData
				tm.mu.RUnlock()

				if onData != nil {
					onData(tunnel.RequestID, payload)
				}

				// Also deliver to channel for sync reads
				select {
				case tunnel.dataChan <- payload:
				default:
					// Buffer full
				}
			}
		}
	}
}

// closeTunnel closes a tunnel cleanly
func (tm *TunnelManager) closeTunnel(tunnel *Tunnel, err error) {
	tunnel.mu.Lock()
	if tunnel.state == TunnelStateClosed {
		tunnel.mu.Unlock()
		return
	}

	tunnel.state = TunnelStateClosed
	if tunnel.secureConn != nil {
		tunnel.secureConn.Close()
	}
	close(tunnel.closeCh)
	tunnel.mu.Unlock()

	// Notify callback
	tm.mu.RLock()
	onClosed := tm.onTunnelClosed
	tm.mu.RUnlock()

	if onClosed != nil {
		onClosed(tunnel.RequestID, err)
	}

	// Remove from active tunnels
	tm.mu.Lock()
	delete(tm.tunnels, tunnel.RequestID)
	tm.mu.Unlock()
}

// GetTunnel returns a tunnel by request ID
func (tm *TunnelManager) GetTunnel(requestID uint32) *Tunnel {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tunnels[requestID]
}

// SendData sends data over a tunnel
func (tm *TunnelManager) SendData(requestID uint32, data []byte) error {
	tunnel := tm.GetTunnel(requestID)
	if tunnel == nil {
		return errors.New("tunnel not found")
	}

	tunnel.mu.RLock()
	conn := tunnel.secureConn
	state := tunnel.state
	tunnel.mu.RUnlock()

	if state != TunnelStateEstablished || conn == nil {
		return errors.New("tunnel not established")
	}

	_, err := conn.Write(data)
	return err
}

// CloseTunnel closes a specific tunnel
func (tm *TunnelManager) CloseTunnel(requestID uint32) error {
	tunnel := tm.GetTunnel(requestID)
	if tunnel == nil {
		return errors.New("tunnel not found")
	}

	tm.closeTunnel(tunnel, nil)
	return nil
}

// CloseAll closes all active tunnels
func (tm *TunnelManager) CloseAll() {
	tm.mu.Lock()
	tunnels := make([]*Tunnel, 0, len(tm.tunnels))
	for _, t := range tm.tunnels {
		tunnels = append(tunnels, t)
	}
	tm.mu.Unlock()

	for _, t := range tunnels {
		tm.closeTunnel(t, nil)
	}
}

// Tunnel methods

// State returns the tunnel state
func (t *Tunnel) State() TunnelState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state
}

// LastError returns the last error that occurred
func (t *Tunnel) LastError() error {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastError
}

// Read reads data from the tunnel (blocking)
func (t *Tunnel) Read() ([]byte, error) {
	select {
	case data := <-t.dataChan:
		return data, nil
	case <-t.closeCh:
		return nil, ErrClosed
	}
}

// Write writes data to the tunnel
func (t *Tunnel) Write(data []byte) error {
	t.mu.RLock()
	conn := t.secureConn
	state := t.state
	t.mu.RUnlock()

	if state != TunnelStateEstablished || conn == nil {
		return ErrClosed
	}

	_, err := conn.Write(data)
	return err
}
