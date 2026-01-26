// Package udp implements the MS-RDPEUDP transport layer for RDP over UDP.
// This file implements the DTLS/TLS security wrapper for secure UDP tunnels.
//
// Reference: [MS-RDPEMT] Section 1.3
// - Reliable UDP (RDP-UDP-R): Uses TLS for security
// - Lossy UDP (RDP-UDP-L): Uses DTLS for security
//
// The security handshake occurs AFTER the RDPEUDP handshake completes.
package udp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/rcarmo/go-rdp/internal/protocol/rdpemt"
)

// SecureConnection wraps an RDPEUDP connection with TLS or DTLS security.
// Per MS-RDPEMT Section 1.3:
// - TLS is used for reliable transport (RDP-UDP-R)
// - DTLS is used for lossy transport (RDP-UDP-L)
type SecureConnection struct {
	mu sync.RWMutex

	// Underlying RDPEUDP connection
	udpConn *Connection

	// Security layer (either tls.Conn or dtls.Conn)
	secureConn io.ReadWriteCloser

	// Whether this is reliable (TLS) or lossy (DTLS)
	reliable bool

	// TLS configuration
	tlsConfig *tls.Config

	// DTLS configuration
	dtlsConfig *dtls.Config

	// Tunnel state
	tunnelEstablished bool
	requestID         uint32
	securityCookie    [16]byte
}

// SecureConfig holds configuration for secure UDP connection
type SecureConfig struct {
	// UDP connection configuration
	UDPConfig *Config

	// Reliable mode (TLS) vs Lossy mode (DTLS)
	Reliable bool

	// TLS configuration for reliable mode
	TLSConfig *tls.Config

	// DTLS configuration for lossy mode
	DTLSConfig *dtls.Config

	// Multitransport request info from server
	RequestID      uint32
	SecurityCookie [16]byte
}

// NewSecureConnection creates a new secure UDP connection
func NewSecureConnection(config *SecureConfig) (*SecureConnection, error) {
	if config == nil {
		return nil, errors.New("secure: config required")
	}

	// Create underlying UDP connection
	udpConn, err := NewConnection(config.UDPConfig)
	if err != nil {
		return nil, fmt.Errorf("secure: create UDP connection: %w", err)
	}

	sc := &SecureConnection{
		udpConn:        udpConn,
		reliable:       config.Reliable,
		tlsConfig:      config.TLSConfig,
		dtlsConfig:     config.DTLSConfig,
		requestID:      config.RequestID,
		securityCookie: config.SecurityCookie,
	}

	// Set defaults for TLS/DTLS if not provided
	if sc.reliable && sc.tlsConfig == nil {
		sc.tlsConfig = &tls.Config{
			InsecureSkipVerify: true, // RDP typically uses self-signed certs
			MinVersion:         tls.VersionTLS12,
		}
	}

	if !sc.reliable && sc.dtlsConfig == nil {
		sc.dtlsConfig = &dtls.Config{
			InsecureSkipVerify: true, // RDP typically uses self-signed certs
		}
	}

	return sc, nil
}

// Connect establishes the secure UDP tunnel.
// Process per MS-RDPEMT:
// 1. Establish RDPEUDP connection (SYN/SYN+ACK/ACK)
// 2. Perform TLS/DTLS handshake over RDPEUDP
// 3. Send Tunnel Create Request with RequestID and SecurityCookie
// 4. Receive Tunnel Create Response
func (sc *SecureConnection) Connect(ctx context.Context) error {
	// Step 1: Establish RDPEUDP connection
	if err := sc.udpConn.Connect(ctx); err != nil {
		return fmt.Errorf("secure: RDPEUDP connect: %w", err)
	}

	// Step 2: Perform security handshake (TLS or DTLS)
	if err := sc.performSecurityHandshake(ctx); err != nil {
		sc.udpConn.Close()
		return fmt.Errorf("secure: security handshake: %w", err)
	}

	// Step 3: Send Tunnel Create Request
	if err := sc.sendTunnelCreateRequest(); err != nil {
		sc.Close()
		return fmt.Errorf("secure: tunnel create request: %w", err)
	}

	// Step 4: Receive Tunnel Create Response
	if err := sc.receiveTunnelCreateResponse(ctx); err != nil {
		sc.Close()
		return fmt.Errorf("secure: tunnel create response: %w", err)
	}

	sc.mu.Lock()
	sc.tunnelEstablished = true
	sc.mu.Unlock()

	return nil
}

// performSecurityHandshake performs TLS or DTLS handshake over RDPEUDP
func (sc *SecureConnection) performSecurityHandshake(ctx context.Context) error {
	if sc.reliable {
		return sc.performTLSHandshake(ctx)
	}
	return sc.performDTLSHandshake(ctx)
}

// performTLSHandshake wraps the UDP connection with TLS for reliable transport
func (sc *SecureConnection) performTLSHandshake(ctx context.Context) error {
	// Wrap RDPEUDP connection as a net.Conn for TLS
	wrapper := &udpConnWrapper{conn: sc.udpConn}

	// Create TLS client connection
	tlsConn := tls.Client(wrapper, sc.tlsConfig)

	// Set deadline from context
	if deadline, ok := ctx.Deadline(); ok {
		tlsConn.SetDeadline(deadline)
	}

	// Perform handshake
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("TLS handshake: %w", err)
	}

	// Clear deadline
	tlsConn.SetDeadline(time.Time{})

	sc.mu.Lock()
	sc.secureConn = tlsConn
	sc.mu.Unlock()

	return nil
}

// performDTLSHandshake wraps the UDP connection with DTLS for lossy transport
func (sc *SecureConnection) performDTLSHandshake(ctx context.Context) error {
	// Wrap RDPEUDP connection as a net.Conn for DTLS
	wrapper := &udpConnWrapper{conn: sc.udpConn}

	if deadline, ok := ctx.Deadline(); ok {
		_ = wrapper.SetDeadline(deadline)
	}

	dtlsConn, err := dtls.Client(wrapper, sc.dtlsConfig)
	if err != nil {
		return fmt.Errorf("DTLS handshake: %w", err)
	}

	sc.mu.Lock()
	sc.secureConn = dtlsConn
	sc.mu.Unlock()

	return nil
}

// sendTunnelCreateRequest sends RDP_TUNNEL_CREATEREQUEST
// Per MS-RDPEMT Section 2.2.2.1
func (sc *SecureConnection) sendTunnelCreateRequest() error {
	// Build Tunnel Create Request
	req := &rdpemt.TunnelCreateRequest{
		RequestID:      sc.requestID,
		Reserved:       0,
		SecurityCookie: sc.securityCookie,
	}

	payload, err := req.Serialize()
	if err != nil {
		return fmt.Errorf("serialize tunnel request: %w", err)
	}

	// Wrap in tunnel header
	header := &rdpemt.TunnelHeader{
		Action:        rdpemt.ActionCreateRequest,
		Flags:         0,
		PayloadLength: uint16(len(payload)),
	}

	headerBytes, err := header.Serialize()
	if err != nil {
		return fmt.Errorf("serialize tunnel header: %w", err)
	}

	// Combine and send
	pdu := append(headerBytes, payload...)

	sc.mu.RLock()
	conn := sc.secureConn
	sc.mu.RUnlock()

	if conn == nil {
		return errors.New("secure connection not established")
	}

	_, err = conn.Write(pdu)
	return err
}

// receiveTunnelCreateResponse reads and validates RDP_TUNNEL_CREATERESPONSE
// Per MS-RDPEMT Section 2.2.2.2
func (sc *SecureConnection) receiveTunnelCreateResponse(ctx context.Context) error {
	sc.mu.RLock()
	conn := sc.secureConn
	sc.mu.RUnlock()

	if conn == nil {
		return errors.New("secure connection not established")
	}

	// Read response (with timeout from context)
	buf := make([]byte, 256)

	// Set read deadline if context has one
	if deadline, ok := ctx.Deadline(); ok {
		if setter, ok := conn.(interface{ SetReadDeadline(time.Time) error }); ok {
			setter.SetReadDeadline(deadline)
		}
	}

	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("read tunnel response: %w", err)
	}

	// Clear deadline
	if setter, ok := conn.(interface{ SetReadDeadline(time.Time) error }); ok {
		setter.SetReadDeadline(time.Time{})
	}

	// Parse tunnel header
	action, payload, err := rdpemt.ParseTunnelPDU(buf[:n])
	if err != nil {
		return fmt.Errorf("parse tunnel response: %w", err)
	}

	if action != rdpemt.ActionCreateResponse {
		return fmt.Errorf("unexpected action: %d, expected CreateResponse", action)
	}

	// Parse response
	resp := &rdpemt.TunnelCreateResponse{}
	if err := resp.Deserialize(payload); err != nil {
		return fmt.Errorf("deserialize tunnel response: %w", err)
	}

	// Check result
	if resp.HResult != rdpemt.HResultSuccess {
		return fmt.Errorf("tunnel creation failed: %s", rdpemt.HResultString(resp.HResult))
	}

	return nil
}

// Read reads data from the secure tunnel
func (sc *SecureConnection) Read(b []byte) (int, error) {
	sc.mu.RLock()
	conn := sc.secureConn
	established := sc.tunnelEstablished
	sc.mu.RUnlock()

	if !established || conn == nil {
		return 0, ErrClosed
	}

	return conn.Read(b)
}

// Write writes data over the secure tunnel wrapped in TunnelDataPDU
func (sc *SecureConnection) Write(b []byte) (int, error) {
	sc.mu.RLock()
	conn := sc.secureConn
	established := sc.tunnelEstablished
	sc.mu.RUnlock()

	if !established || conn == nil {
		return 0, ErrClosed
	}

	// Wrap data in TunnelDataPDU
	pdu := &rdpemt.TunnelDataPDU{
		Data: b,
	}

	data, err := pdu.Serialize()
	if err != nil {
		return 0, fmt.Errorf("serialize tunnel data: %w", err)
	}

	_, err = conn.Write(data)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

// Close closes the secure connection
func (sc *SecureConnection) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	var errs []error

	if sc.secureConn != nil {
		if err := sc.secureConn.Close(); err != nil {
			errs = append(errs, err)
		}
		sc.secureConn = nil
	}

	if sc.udpConn != nil {
		if err := sc.udpConn.Close(); err != nil {
			errs = append(errs, err)
		}
		sc.udpConn = nil
	}

	sc.tunnelEstablished = false

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// LocalAddr returns the local network address
func (sc *SecureConnection) LocalAddr() net.Addr {
	if sc.udpConn != nil {
		return sc.udpConn.LocalAddr()
	}
	return nil
}

// RemoteAddr returns the remote network address
func (sc *SecureConnection) RemoteAddr() net.Addr {
	if sc.udpConn != nil {
		return sc.udpConn.RemoteAddr()
	}
	return nil
}

// IsEstablished returns true if the secure tunnel is established
func (sc *SecureConnection) IsEstablished() bool {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.tunnelEstablished
}

// udpConnWrapper wraps a UDP Connection to implement net.Conn interface
// This allows TLS to work over our RDPEUDP connection
type udpConnWrapper struct {
	conn *Connection
}

func (w *udpConnWrapper) Read(b []byte) (int, error) {
	return w.conn.Read(b)
}

func (w *udpConnWrapper) Write(b []byte) (int, error) {
	return w.conn.Write(b)
}

func (w *udpConnWrapper) Close() error {
	return w.conn.Close()
}

func (w *udpConnWrapper) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *udpConnWrapper) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *udpConnWrapper) SetDeadline(t time.Time) error {
	// Deadlines are handled at the UDP level
	return nil
}

func (w *udpConnWrapper) SetReadDeadline(t time.Time) error {
	return nil
}

func (w *udpConnWrapper) SetWriteDeadline(t time.Time) error {
	return nil
}
