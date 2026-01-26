// Package udp implements the MS-RDPEUDP transport layer for RDP over UDP.
// This provides reliable or lossy UDP transport with optional DTLS encryption.
//
// Reference: [MS-RDPEUDP] Section 3.1.5 - Processing Events and Sequencing Rules
// https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp/
package udp

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/rcarmo/go-rdp/internal/protocol/rdpeudp"
)

// Connection states per MS-RDPEUDP Section 3.1.5
// State diagram: https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-rdpeudp/575660d7-a698-48de-92db-d4d4c9fcc783
type State int

const (
	// StateClosed - Endpoint is closed, doesn't respond to any events
	StateClosed State = iota
	// StateListen - Server only: listening for incoming connections
	StateListen
	// StateSynSent - Client only: SYN packet sent, waiting for SYN+ACK
	StateSynSent
	// StateSynReceived - Server only: SYN received, SYN+ACK sent
	StateSynReceived
	// StateEstablished - Connection established, data can be exchanged
	StateEstablished
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateListen:
		return "LISTEN"
	case StateSynSent:
		return "SYN_SENT"
	case StateSynReceived:
		return "SYN_RECEIVED"
	case StateEstablished:
		return "ESTABLISHED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}

// Protocol constants per MS-RDPEUDP specification
const (
	// DefaultMTU is the default MTU value (range: 1132-1232)
	DefaultMTU = 1232

	// MinMTU is the minimum allowed MTU
	MinMTU = 1132

	// MaxMTU is the maximum allowed MTU
	MaxMTU = 1232

	// DefaultReceiveWindowSize is the default receive buffer size in packets
	DefaultReceiveWindowSize = 64

	// MaxRetransmitCount is the max SYN/SYN+ACK retransmit attempts (3-5 per spec)
	MaxRetransmitCount = 3

	// MaxDataRetransmitCount is max data packet retransmit attempts before close
	// Per MS-RDPEUDP Section 3.1.6.1: "at least three and no more than five times"
	MaxDataRetransmitCount = 5

	// RetransmitTimeoutV1 is the minimum retransmit timeout for protocol version 1
	// Per MS-RDPEUDP Section 3.1.6.1: "500 ms"
	RetransmitTimeoutV1 = 500 * time.Millisecond

	// RetransmitTimeoutV2 is the minimum retransmit timeout for protocol version 2+
	// Per MS-RDPEUDP Section 3.1.6.1: "300 ms"
	RetransmitTimeoutV2 = 300 * time.Millisecond

	// KeepaliveTimeout is the timeout before connection is considered dead
	// Per MS-RDPEUDP Section 3.1.1.9: "65 seconds"
	KeepaliveTimeout = 65 * time.Second

	// KeepaliveInterval is how often to send keepalive ACKs
	// Should be less than KeepaliveTimeout to keep NAT mappings alive
	KeepaliveInterval = 30 * time.Second

	// DelayedACKTimeout is the max delay before sending a pending ACK
	// Per MS-RDPEUDP Section 3.1.6.3
	DelayedACKTimeout = 200 * time.Millisecond

	// ConnectionTimeout is the max time to wait for connection establishment
	ConnectionTimeout = 10 * time.Second

	// LostPacketThreshold is number of later packets received before marking lost
	// Per MS-RDPEUDP Section 3.1.1.4.1: "three other datagrams"
	LostPacketThreshold = 3
)

// ACK vector element states per MS-RDPEUDP Section 2.2.1.1
const (
	AckStateReceived   uint8 = 0 // DATAGRAM_RECEIVED
	AckStateReserved1  uint8 = 1 // Not used
	AckStateReserved2  uint8 = 2 // Not used
	AckStateNotReceived uint8 = 3 // DATAGRAM_NOT_YET_RECEIVED
)

// Errors
var (
	ErrClosed           = errors.New("udp: connection closed")
	ErrTimeout          = errors.New("udp: connection timeout")
	ErrInvalidState     = errors.New("udp: invalid state for operation")
	ErrInvalidPacket    = errors.New("udp: invalid packet")
	ErrNotImplemented   = errors.New("udp: not implemented")
	ErrConnectionFailed = errors.New("udp: connection establishment failed")
)

// Config holds UDP connection configuration
type Config struct {
	// LocalAddr is the local address to bind to (optional for client)
	LocalAddr *net.UDPAddr

	// RemoteAddr is the remote address to connect to (required for client)
	RemoteAddr *net.UDPAddr

	// MTU is the maximum transmission unit (1132-1232)
	MTU uint16

	// ReceiveWindowSize is the receive buffer size in packets
	ReceiveWindowSize uint16

	// Reliable enables reliable transport mode (RDP-UDP-R)
	// If false, uses lossy transport mode (RDP-UDP-L)
	Reliable bool

	// CookieHash is the SHA-256 hash of the security cookie (for version 3)
	CookieHash [32]byte

	// ProtocolVersion is the RDPEUDP protocol version to use
	// 0x0001 = Version 1, 0x0002 = Version 2, 0x0101 = Version 3
	ProtocolVersion uint16
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		MTU:               DefaultMTU,
		ReceiveWindowSize: DefaultReceiveWindowSize,
		Reliable:          true,
		ProtocolVersion:   rdpeudp.ProtocolVersion2,
	}
}

// Connection represents an RDPEUDP connection
type Connection struct {
	mu sync.RWMutex

	// Configuration
	config *Config

	// UDP socket
	conn *net.UDPConn

	// Connection state
	state State

	// Sequence numbers
	localSeqNum   uint32 // Our initial sequence number
	remoteSeqNum  uint32 // Peer's initial sequence number
	nextSendSeq   uint32 // Next sequence number to send
	nextExpectSeq uint32 // Next sequence number we expect to receive

	// Acknowledgment tracking
	lastAckedSeq   uint32 // Last sequence number acknowledged by peer
	highestRecvSeq uint32 // Highest sequence number received (for ACK vector)
	pendingAck     bool   // Whether we have a pending ACK to send

	// MTU negotiation results
	upstreamMTU   uint16
	downstreamMTU uint16

	// Protocol version negotiated
	negotiatedVersion uint16

	// Retransmission state
	synRetryCount int
	lastSendTime  time.Time
	lastRecvTime  time.Time // For keepalive tracking
	rtt           time.Duration // Round-trip time estimate

	// Congestion control
	congestionWindow int      // Current congestion window size
	congestionNotify bool     // Need to send CN flag

	// Receive buffer for out-of-order packets
	recvBuffer map[uint32][]byte

	// Send buffer for retransmission
	sendBuffer map[uint32]*sentPacket

	// Channels
	recvChan    chan []byte
	closeChan   chan struct{}
	closedOnce  sync.Once
	established chan struct{}

	// Timers
	keepaliveTimer    *time.Timer
	retransmitTimer   *time.Timer
	delayedAckTimer   *time.Timer

	// Statistics
	stats ConnectionStats
}

// sentPacket tracks a packet waiting for acknowledgment
type sentPacket struct {
	data       []byte
	seqNum     uint32
	sentTime   time.Time
	retryCount int
	nextRetry  time.Time // When to next retry
}

// ConnectionStats holds connection statistics
type ConnectionStats struct {
	PacketsSent       uint64
	PacketsReceived   uint64
	BytesSent         uint64
	BytesReceived     uint64
	Retransmits       uint64
	PacketsLost       uint64
	RTT               time.Duration // Current RTT estimate
	CongestionEvents  uint64
}

// NewConnection creates a new RDPEUDP connection
func NewConnection(config *Config) (*Connection, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Validate MTU
	if config.MTU < MinMTU || config.MTU > MaxMTU {
		config.MTU = DefaultMTU
	}
	if config.MTU == 0 {
		config.MTU = DefaultMTU
	}

	c := &Connection{
		config:           config,
		state:            StateClosed,
		recvBuffer:       make(map[uint32][]byte),
		sendBuffer:       make(map[uint32]*sentPacket),
		recvChan:         make(chan []byte, 256),
		closeChan:        make(chan struct{}),
		established:      make(chan struct{}),
		rtt:              RetransmitTimeoutV2, // Initial estimate
		congestionWindow: 16, // Initial cwnd
	}

	// Generate random initial sequence number per spec Section 3.1.5.1.1
	c.localSeqNum = generateInitialSequenceNumber()
	c.nextSendSeq = c.localSeqNum

	return c, nil
}

// generateInitialSequenceNumber generates a random 32-bit sequence number
// Per MS-RDPEUDP Section 3.1.5.1.1: "snInitialSequenceNumber variable MUST be set
// to a 32-bit number generated by using a truly random function"
func generateInitialSequenceNumber() uint32 {
	max := new(big.Int).Lsh(big.NewInt(1), 32)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// Fallback to time-based if crypto/rand fails
		return uint32(time.Now().UnixNano())
	}
	return uint32(n.Uint64())
}

// State returns the current connection state
func (c *Connection) State() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// Stats returns connection statistics
func (c *Connection) Stats() ConnectionStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// Connect initiates a connection to the remote server (client-side)
// Per MS-RDPEUDP Section 3.1.5: Connection Sequence
func (c *Connection) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.state != StateClosed {
		c.mu.Unlock()
		return ErrInvalidState
	}

	if c.config.RemoteAddr == nil {
		c.mu.Unlock()
		return errors.New("udp: remote address required for Connect")
	}

	// Create UDP socket
	var err error
	c.conn, err = net.DialUDP("udp", c.config.LocalAddr, c.config.RemoteAddr)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("udp: dial failed: %w", err)
	}

	c.state = StateSynSent
	c.mu.Unlock()

	// Start receive goroutine
	go c.receiveLoop()

	// Send SYN and wait for SYN+ACK
	if err := c.sendSyn(ctx); err != nil {
		c.Close()
		return err
	}

	// Wait for established state or timeout
	select {
	case <-c.established:
		return nil
	case <-ctx.Done():
		c.Close()
		return ErrTimeout
	case <-c.closeChan:
		return ErrClosed
	}
}

// sendSyn sends a SYN packet and handles retransmission
// Per MS-RDPEUDP Section 3.1.5.1.1
func (c *Connection) sendSyn(ctx context.Context) error {
	synPacket := c.buildSynPacket()

	// Get version-appropriate timeout
	baseTimeout := c.getRetransmitTimeout()

	for c.synRetryCount < MaxRetransmitCount {
		c.mu.Lock()
		if c.state != StateSynSent {
			c.mu.Unlock()
			return nil // State changed (connection established or closed)
		}
		c.mu.Unlock()

		// Send SYN
		if err := c.sendPacket(synPacket); err != nil {
			return err
		}

		c.mu.Lock()
		c.synRetryCount++
		c.lastSendTime = time.Now()
		c.mu.Unlock()

		// Wait for SYN+ACK or timeout with exponential backoff
		timeout := baseTimeout * time.Duration(1<<uint(c.synRetryCount-1))
		timer := time.NewTimer(timeout)
		select {
		case <-c.established:
			timer.Stop()
			return nil
		case <-timer.C:
			// Timeout, retry
			continue
		case <-ctx.Done():
			timer.Stop()
			return ErrTimeout
		case <-c.closeChan:
			timer.Stop()
			return ErrClosed
		}
	}

	return ErrConnectionFailed
}

// getRetransmitTimeout returns the minimum retransmit timeout based on protocol version
// Per MS-RDPEUDP Section 3.1.6.1
func (c *Connection) getRetransmitTimeout() time.Duration {
	c.mu.RLock()
	version := c.config.ProtocolVersion
	rtt := c.rtt
	c.mu.RUnlock()

	var minTimeout time.Duration
	if version >= rdpeudp.ProtocolVersion2 {
		minTimeout = RetransmitTimeoutV2
	} else {
		minTimeout = RetransmitTimeoutV1
	}

	// Per spec: "minimum retransmit time-out or twice the RTT, whichever is longer"
	rttTimeout := 2 * rtt
	if rttTimeout > minTimeout {
		return rttTimeout
	}
	return minTimeout
}

// buildSynPacket constructs a SYN packet per MS-RDPEUDP Section 3.1.5.1.1
func (c *Connection) buildSynPacket() *rdpeudp.Packet {
	packet := rdpeudp.NewSYNPacket(
		c.localSeqNum,
		c.config.MTU,
		c.config.MTU,
		c.config.ReceiveWindowSize,
	)

	// Add SYNLOSSY flag for lossy mode
	if !c.config.Reliable {
		packet.Header.Flags |= rdpeudp.FlagSYNLOSSY
	}

	// Add SYNEX payload for version negotiation
	if c.config.ProtocolVersion >= rdpeudp.ProtocolVersion2 {
		packet.Header.Flags |= rdpeudp.FlagSYNEX
		packet.SynDataEx = &rdpeudp.SynDataEx{
			Flags:   rdpeudp.SynExFlagVersionInfoValid,
			Version: c.config.ProtocolVersion,
		}

		// Add cookie hash for version 3
		if c.config.ProtocolVersion == rdpeudp.ProtocolVersion3 {
			packet.SynDataEx.CookieHash = c.config.CookieHash
		}
	}

	return packet
}

// handleReceivedPacket processes an incoming packet
func (c *Connection) handleReceivedPacket(data []byte) {
	packet := &rdpeudp.Packet{}
	if err := packet.Deserialize(data); err != nil {
		// Invalid packet, ignore
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.stats.PacketsReceived++
	c.stats.BytesReceived += uint64(len(data))
	c.lastRecvTime = time.Now() // Update keepalive timestamp

	// Reset keepalive timer
	c.resetKeepaliveTimer()

	switch c.state {
	case StateSynSent:
		c.handleSynSentState(packet)
	case StateEstablished:
		c.handleEstablishedState(packet)
	}
}

// handleSynSentState handles packets in SYN_SENT state
// Expecting SYN+ACK from server
func (c *Connection) handleSynSentState(packet *rdpeudp.Packet) {
	// Must have both SYN and ACK flags
	if !packet.Header.HasFlag(rdpeudp.FlagSYN) || !packet.Header.HasFlag(rdpeudp.FlagACK) {
		return
	}

	// Verify ACK is for our SYN
	// Per spec: snSourceAck should equal our snInitialSequenceNumber
	if packet.Header.SnSourceAck != c.localSeqNum {
		return
	}

	// Calculate initial RTT estimate from SYN-SYN/ACK timing
	if !c.lastSendTime.IsZero() {
		c.rtt = time.Since(c.lastSendTime)
		c.stats.RTT = c.rtt
	}

	// Store remote sequence number
	if packet.SynData != nil {
		c.remoteSeqNum = packet.SynData.SnInitialSequenceNumber
		c.nextExpectSeq = c.remoteSeqNum + 1
		c.highestRecvSeq = c.remoteSeqNum

		// Negotiate MTU (minimum of both)
		c.upstreamMTU = minUint16(c.config.MTU, packet.SynData.UpstreamMTU)
		c.downstreamMTU = minUint16(c.config.MTU, packet.SynData.DownstreamMTU)
	}

	// Negotiate protocol version
	if packet.SynDataEx != nil && packet.SynDataEx.Flags&rdpeudp.SynExFlagVersionInfoValid != 0 {
		c.negotiatedVersion = minUint16(c.config.ProtocolVersion, packet.SynDataEx.Version)
	} else {
		c.negotiatedVersion = rdpeudp.ProtocolVersion1
	}

	// Send ACK to complete handshake
	c.state = StateEstablished
	c.nextSendSeq = c.localSeqNum + 1

	// Signal established
	close(c.established)

	// Start keepalive timer
	c.startKeepaliveTimer()

	// Send ACK packet
	go c.sendAck()
}

// handleEstablishedState handles packets in ESTABLISHED state
func (c *Connection) handleEstablishedState(packet *rdpeudp.Packet) {
	// Process congestion notification
	if packet.Header.HasFlag(rdpeudp.FlagCN) {
		// Peer detected congestion - reduce our send rate
		c.handleCongestionNotification()
	}

	// Process congestion window reset acknowledgment
	if packet.Header.HasFlag(rdpeudp.FlagCWR) {
		// Peer acknowledged our CN, stop sending CN flags
		c.congestionNotify = false
	}

	// Process ACK
	if packet.Header.HasFlag(rdpeudp.FlagACK) {
		c.processAck(packet)
	}

	// Process data
	if packet.Header.HasFlag(rdpeudp.FlagDAT) && packet.SourcePayload != nil {
		c.processData(packet)
	}

	// Process FIN
	if packet.Header.HasFlag(rdpeudp.FlagFIN) {
		c.state = StateClosed
		c.closedOnce.Do(func() { close(c.closeChan) })
	}
}

// handleCongestionNotification handles CN flag from peer
// Per MS-RDPEUDP Section 3.1.1.6
func (c *Connection) handleCongestionNotification() {
	// Reduce congestion window (multiplicative decrease)
	c.congestionWindow = c.congestionWindow / 2
	if c.congestionWindow < 1 {
		c.congestionWindow = 1
	}
	c.stats.CongestionEvents++
}

// processAck processes acknowledgment in received packet
func (c *Connection) processAck(packet *rdpeudp.Packet) {
	ackSeq := packet.Header.SnSourceAck
	if ackSeq > c.lastAckedSeq {
		c.lastAckedSeq = ackSeq

		// Remove acknowledged packets from send buffer
		for seq := range c.sendBuffer {
			if seq <= ackSeq {
				delete(c.sendBuffer, seq)
			}
		}
	}

	// Process ACK vector for selective ACK
	if packet.AckVector != nil {
		c.processAckVector(packet.AckVector)
	}
}

// processAckVector processes selective ACK information
// Per MS-RDPEUDP Section 2.2.2.7 and 3.1.1.4.1
func (c *Connection) processAckVector(ackVector *rdpeudp.AckVector) {
	if ackVector == nil || len(ackVector.AckVectorElements) == 0 {
		return
	}

	// ACK vector is RLE-encoded starting from snSourceAck and going backwards
	// Each element: 2 bits state, 6 bits length
	currentSeq := c.lastAckedSeq
	lostPackets := make([]uint32, 0)

	for _, element := range ackVector.AckVectorElements {
		state := (element >> 6) & 0x03  // Top 2 bits
		length := int(element & 0x3F)   // Bottom 6 bits (0-63)

		// Process 'length + 1' packets with this state
		for i := 0; i <= length; i++ {
			switch state {
			case AckStateReceived:
				// Packet received, remove from send buffer
				delete(c.sendBuffer, currentSeq)
			case AckStateNotReceived:
				// Packet not received, schedule retransmission
				if pkt, ok := c.sendBuffer[currentSeq]; ok {
					lostPackets = append(lostPackets, pkt.seqNum)
				}
			}
			currentSeq-- // ACK vector goes backwards
		}
	}

	// Retransmit lost packets
	for _, seqNum := range lostPackets {
		c.retransmitPacket(seqNum)
	}
}

// retransmitPacket retransmits a packet by sequence number
func (c *Connection) retransmitPacket(seqNum uint32) {
	pkt, ok := c.sendBuffer[seqNum]
	if !ok {
		return
	}

	// Check retry count per MS-RDPEUDP Section 3.1.6.1
	if pkt.retryCount >= MaxDataRetransmitCount {
		// Connection should be terminated
		c.state = StateClosed
		c.closedOnce.Do(func() { close(c.closeChan) })
		return
	}

	pkt.retryCount++
	pkt.sentTime = time.Now()
	pkt.nextRetry = time.Now().Add(c.getRetransmitTimeout())
	c.stats.Retransmits++

	// Re-send the packet data
	if c.conn != nil && len(pkt.data) > 0 {
		c.conn.Write(pkt.data)
	}
}

// processData processes incoming data payload
func (c *Connection) processData(packet *rdpeudp.Packet) {
	if packet.SourcePayload == nil {
		return
	}

	seqNum := packet.SourcePayload.SnSourceStart

	// Track highest received for ACK vector generation
	if seqNum > c.highestRecvSeq {
		// Detect gaps (potential packet loss)
		if seqNum > c.highestRecvSeq+1 {
			// We missed some packets - set congestion notify flag
			c.congestionNotify = true
		}
		c.highestRecvSeq = seqNum
	}

	// Mark that we have data to acknowledge
	c.pendingAck = true

	// Check if this is the next expected packet
	if seqNum == c.nextExpectSeq {
		// Deliver data
		select {
		case c.recvChan <- packet.Data:
		default:
			// Buffer full, drop packet
			c.stats.PacketsLost++
		}
		c.nextExpectSeq++

		// Check for buffered out-of-order packets
		for {
			if data, ok := c.recvBuffer[c.nextExpectSeq]; ok {
				select {
				case c.recvChan <- data:
				default:
					c.stats.PacketsLost++
				}
				delete(c.recvBuffer, c.nextExpectSeq)
				c.nextExpectSeq++
			} else {
				break
			}
		}
	} else if seqNum > c.nextExpectSeq {
		// Out of order, buffer it
		c.recvBuffer[seqNum] = packet.Data
	}
	// If seqNum < nextExpectSeq, it's a duplicate, ignore

	// Start delayed ACK timer if not already running
	c.startDelayedAckTimer()
}

// sendAck sends an ACK packet
// Per MS-RDPEUDP Section 3.1.5.2.1
func (c *Connection) sendAck() error {
	c.mu.Lock()
	ackPacket := c.buildAckPacket()
	c.pendingAck = false
	c.mu.Unlock()

	return c.sendPacket(ackPacket)
}

// buildAckPacket constructs an ACK packet with current state
func (c *Connection) buildAckPacket() *rdpeudp.Packet {
	packet := rdpeudp.NewACKPacket(
		c.highestRecvSeq, // ACK the highest received sequence
		c.config.ReceiveWindowSize,
	)

	// Add congestion notification if we detected loss
	// Per MS-RDPEUDP Section 2.2.2.1
	if c.congestionNotify {
		packet.Header.Flags |= rdpeudp.FlagCN
	}

	// Build ACK vector for selective acknowledgment
	packet.AckVector = c.buildAckVector()

	return packet
}

// buildAckVector builds an RLE-encoded ACK vector
// Per MS-RDPEUDP Section 3.1.1.4.1
func (c *Connection) buildAckVector() *rdpeudp.AckVector {
	if c.highestRecvSeq == 0 || c.nextExpectSeq == 0 {
		return nil
	}

	// Calculate how many packets to represent
	// From highestRecvSeq going backwards to nextExpectSeq
	if c.highestRecvSeq < c.nextExpectSeq {
		return nil
	}

	elements := make([]uint8, 0)
	currentState := AckStateReceived
	runLength := 0

	// Go backwards from highestRecvSeq
	for seq := c.highestRecvSeq; seq >= c.nextExpectSeq && len(elements) < 2048; seq-- {
		var state uint8
		if seq < c.nextExpectSeq {
			// Already received and delivered
			state = AckStateReceived
		} else if _, ok := c.recvBuffer[seq]; ok {
			// Buffered (received out of order)
			state = AckStateReceived
		} else if seq == c.highestRecvSeq || seq < c.nextExpectSeq {
			state = AckStateReceived
		} else {
			// Not received
			state = AckStateNotReceived
		}

		if state == currentState && runLength < 63 {
			runLength++
		} else {
			// Flush current run
			if runLength > 0 {
				element := (currentState << 6) | uint8(runLength-1)
				elements = append(elements, element)
			}
			currentState = state
			runLength = 1
		}

		// Prevent underflow
		if seq == 0 {
			break
		}
	}

	// Flush final run
	if runLength > 0 {
		element := (currentState << 6) | uint8(runLength-1)
		elements = append(elements, element)
	}

	if len(elements) == 0 {
		return nil
	}

	return &rdpeudp.AckVector{
		AckVectorSize:     uint16(len(elements)),
		AckVectorElements: elements,
	}
}

// sendPacket serializes and sends a packet
func (c *Connection) sendPacket(packet *rdpeudp.Packet) error {
	data, err := packet.Serialize()
	if err != nil {
		return err
	}

	// Pad to MTU for SYN packets per spec
	if packet.Header.HasFlag(rdpeudp.FlagSYN) {
		c.mu.RLock()
		mtu := int(c.config.MTU)
		c.mu.RUnlock()
		if len(data) < mtu {
			padding := make([]byte, mtu-len(data))
			data = append(data, padding...)
		}
	}

	c.mu.Lock()
	c.stats.PacketsSent++
	c.stats.BytesSent += uint64(len(data))
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return nil // No socket, skip sending
	}

	_, err = conn.Write(data)
	return err
}

// receiveLoop reads packets from the UDP socket
func (c *Connection) receiveLoop() {
	buf := make([]byte, 2048)
	for {
		select {
		case <-c.closeChan:
			return
		default:
		}

		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, err := c.conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			// Connection error
			c.mu.Lock()
			if c.state != StateClosed {
				c.state = StateClosed
				c.closedOnce.Do(func() { close(c.closeChan) })
			}
			c.mu.Unlock()
			return
		}

		if n > 0 {
			// Copy data and process
			data := make([]byte, n)
			copy(data, buf[:n])
			c.handleReceivedPacket(data)
		}
	}
}

// Read reads data from the connection
func (c *Connection) Read(b []byte) (int, error) {
	select {
	case data := <-c.recvChan:
		n := copy(b, data)
		return n, nil
	case <-c.closeChan:
		return 0, ErrClosed
	}
}

// Write sends data over the connection
func (c *Connection) Write(b []byte) (int, error) {
	c.mu.Lock()
	if c.state != StateEstablished {
		c.mu.Unlock()
		return 0, ErrInvalidState
	}

	seqNum := c.nextSendSeq
	c.nextSendSeq++
	c.mu.Unlock()

	packet := rdpeudp.NewDataPacket(seqNum, seqNum, b)

	// Serialize the packet for potential retransmission
	data, err := packet.Serialize()
	if err != nil {
		return 0, err
	}

	// Add to send buffer for potential retransmission
	now := time.Now()
	c.mu.Lock()
	c.sendBuffer[seqNum] = &sentPacket{
		data:      data,
		seqNum:    seqNum,
		sentTime:  now,
		nextRetry: now.Add(c.getRetransmitTimeout()),
	}
	// Start retransmit timer if not running
	c.startRetransmitTimer()
	c.mu.Unlock()

	if err := c.sendPacket(packet); err != nil {
		return 0, err
	}

	return len(b), nil
}

// Close closes the connection
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StateClosed {
		return nil
	}

	// Stop all timers
	c.stopTimers()

	// Send FIN if established
	if c.state == StateEstablished {
		finPacket := rdpeudp.NewFINPacket(c.nextSendSeq)
		go c.sendPacket(finPacket)
	}

	c.state = StateClosed
	c.closedOnce.Do(func() { close(c.closeChan) })

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// LocalAddr returns the local network address
func (c *Connection) LocalAddr() net.Addr {
	if c.conn != nil {
		return c.conn.LocalAddr()
	}
	return nil
}

// RemoteAddr returns the remote network address
func (c *Connection) RemoteAddr() net.Addr {
	if c.conn != nil {
		return c.conn.RemoteAddr()
	}
	return nil
}

// Helper functions

func minUint16(a, b uint16) uint16 {
	if a < b {
		return a
	}
	return b
}

// ReadUint32LE reads a little-endian uint32 from bytes
func ReadUint32LE(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

// Timer management functions

// startKeepaliveTimer starts the keepalive timer
// Per MS-RDPEUDP Section 3.1.1.9 and 3.1.6.2
func (c *Connection) startKeepaliveTimer() {
	if c.keepaliveTimer != nil {
		c.keepaliveTimer.Stop()
	}
	c.keepaliveTimer = time.AfterFunc(KeepaliveInterval, func() {
		c.onKeepaliveTimer()
	})
}

// resetKeepaliveTimer resets the keepalive timer after receiving data
func (c *Connection) resetKeepaliveTimer() {
	if c.keepaliveTimer != nil {
		c.keepaliveTimer.Stop()
		c.keepaliveTimer.Reset(KeepaliveInterval)
	}
}

// onKeepaliveTimer handles keepalive timer expiration
// Per MS-RDPEUDP Section 3.1.6.2: send ACK to keep connection alive
func (c *Connection) onKeepaliveTimer() {
	c.mu.RLock()
	state := c.state
	lastRecv := c.lastRecvTime
	c.mu.RUnlock()

	if state != StateEstablished {
		return
	}

	// Check if we've exceeded the 65-second timeout without receiving anything
	if !lastRecv.IsZero() && time.Since(lastRecv) > KeepaliveTimeout {
		// Connection dead - close it
		c.Close()
		return
	}

	// Send keepalive ACK
	c.sendAck()

	// Restart timer
	c.mu.Lock()
	c.startKeepaliveTimer()
	c.mu.Unlock()
}

// startDelayedAckTimer starts the delayed ACK timer
// Per MS-RDPEUDP Section 3.1.6.3
func (c *Connection) startDelayedAckTimer() {
	if c.delayedAckTimer != nil {
		return // Already running
	}
	c.delayedAckTimer = time.AfterFunc(DelayedACKTimeout, func() {
		c.onDelayedAckTimer()
	})
}

// onDelayedAckTimer handles delayed ACK timer expiration
func (c *Connection) onDelayedAckTimer() {
	c.mu.Lock()
	c.delayedAckTimer = nil
	hasPending := c.pendingAck
	state := c.state
	c.mu.Unlock()

	if hasPending && state == StateEstablished {
		c.sendAck()
	}
}

// startRetransmitTimer starts the retransmit timer for data packets
func (c *Connection) startRetransmitTimer() {
	if c.retransmitTimer != nil {
		return
	}
	timeout := c.getRetransmitTimeout()
	c.retransmitTimer = time.AfterFunc(timeout, func() {
		c.onRetransmitTimer()
	})
}

// onRetransmitTimer handles retransmit timer expiration
// Per MS-RDPEUDP Section 3.1.6.1
func (c *Connection) onRetransmitTimer() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.retransmitTimer = nil
	if c.state != StateEstablished {
		return
	}

	now := time.Now()
	hasOutstanding := false

	// Check all packets in send buffer for retransmission
	for seqNum, pkt := range c.sendBuffer {
		if now.After(pkt.nextRetry) {
			if pkt.retryCount >= MaxDataRetransmitCount {
				// Too many retransmissions - close connection
				c.state = StateClosed
				c.closedOnce.Do(func() { close(c.closeChan) })
				return
			}

			// Retransmit
			pkt.retryCount++
			pkt.sentTime = now
			pkt.nextRetry = now.Add(c.getRetransmitTimeout())
			c.stats.Retransmits++

			if c.conn != nil && len(pkt.data) > 0 {
				c.conn.Write(pkt.data)
			}
		}

		if pkt.seqNum > c.lastAckedSeq {
			hasOutstanding = true
		}
		_ = seqNum // Silence unused warning
	}

	// Restart timer if we still have outstanding packets
	if hasOutstanding {
		c.startRetransmitTimer()
	}
}

// stopTimers stops all running timers
func (c *Connection) stopTimers() {
	if c.keepaliveTimer != nil {
		c.keepaliveTimer.Stop()
		c.keepaliveTimer = nil
	}
	if c.retransmitTimer != nil {
		c.retransmitTimer.Stop()
		c.retransmitTimer = nil
	}
	if c.delayedAckTimer != nil {
		c.delayedAckTimer.Stop()
		c.delayedAckTimer = nil
	}
}
