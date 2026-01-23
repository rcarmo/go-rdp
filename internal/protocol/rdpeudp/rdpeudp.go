// Package rdpeudp implements the RDP UDP Transport Protocol (MS-RDPEUDP).
// This protocol provides reliable and lossy UDP transport for RDP connections.
package rdpeudp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

// Protocol constants from [MS-RDPEUDP]
const (
	// Protocol versions
	Version1 uint8 = 0x01 // Original RDPEUDP
	Version2 uint8 = 0x02 // RDPEUDP2 with enhancements

	// RDPUDP_FLAG values from [MS-RDPEUDP] Section 2.2.1.1
	FlagSYN uint16 = 0x0001 // Synchronization packet
	FlagFIN uint16 = 0x0002 // Finish (connection close)
	FlagACK uint16 = 0x0004 // Acknowledgment
	FlagDAT uint16 = 0x0008 // Data packet
	FlagFEC uint16 = 0x0010 // Forward Error Correction
	FlagCN  uint16 = 0x0020 // Connection correlation
	FlagCWR uint16 = 0x0040 // Congestion Window Reduced
	FlagAOA uint16 = 0x0100 // Ack of Acks
	FlagSYN2 uint16 = 0x0200 // SYN phase 2 (version 2)
	FlagACKV uint16 = 0x0400 // ACK vector present

	// FEC modes
	FECReliable uint8 = 0x01 // Reliable mode with retransmission
	FECLossy    uint8 = 0x02 // Lossy mode for audio/video

	// Default values
	DefaultSnSourceAck    uint32 = 0xFFFFFFFF // Initial ack sequence number
	DefaultReceiveWindow  uint16 = 0x0040     // Default receive window size (64)
	DefaultMTU            uint16 = 1232       // Default MTU for RDPEUDP
	DefaultKeepAliveMs    uint32 = 65000      // Keep-alive interval in milliseconds
	DefaultRetransmitMs   uint32 = 200        // Initial retransmit timeout
	DefaultMaxRetransmits uint8  = 5          // Max retransmission attempts

	// Header sizes
	FECHeaderSize          = 4  // snSourceStart(4)
	AckHeaderSize          = 12 // snAcked(4) + receiveWindow(2) + flags(1) + reserved(1) + ackVector(4)
	SynDataSize            = 8  // snInitialSequenceNumber(4) + upstreamMTU(2) + downstreamMTU(2)
	SourcePayloadHeaderMin = 4  // snSourceStart(4), optional snCoded(4)
)

// Errors
var (
	ErrInvalidPacket        = errors.New("rdpeudp: invalid packet")
	ErrUnsupportedVersion   = errors.New("rdpeudp: unsupported protocol version")
	ErrInvalidFECHeader     = errors.New("rdpeudp: invalid FEC header")
	ErrConnectionReset      = errors.New("rdpeudp: connection reset by peer")
	ErrSequenceNumberWrap   = errors.New("rdpeudp: sequence number wrapped")
	ErrDuplicatePacket      = errors.New("rdpeudp: duplicate packet")
	ErrOutOfWindow          = errors.New("rdpeudp: packet outside receive window")
)

// FECHeader represents the RDPUDP_FEC_HEADER structure.
// [MS-RDPEUDP] Section 2.2.1.1
type FECHeader struct {
	SnSourceAck uint32 // Sequence number being acknowledged
	SourceAckReceiveWindowSize uint16 // Receive window size
	Flags uint16 // RDPUDP_FLAG values
}

// Size returns the serialized size of FECHeader.
func (h *FECHeader) Size() int {
	return 8
}

// Deserialize parses an FECHeader from bytes.
func (h *FECHeader) Deserialize(data []byte) error {
	if len(data) < h.Size() {
		return fmt.Errorf("%w: FEC header too short", ErrInvalidPacket)
	}

	h.SnSourceAck = binary.LittleEndian.Uint32(data[0:4])
	h.SourceAckReceiveWindowSize = binary.LittleEndian.Uint16(data[4:6])
	h.Flags = binary.LittleEndian.Uint16(data[6:8])
	return nil
}

// Serialize encodes an FECHeader to bytes.
func (h *FECHeader) Serialize() []byte {
	buf := make([]byte, h.Size())
	binary.LittleEndian.PutUint32(buf[0:4], h.SnSourceAck)
	binary.LittleEndian.PutUint16(buf[4:6], h.SourceAckReceiveWindowSize)
	binary.LittleEndian.PutUint16(buf[6:8], h.Flags)
	return buf
}

// HasFlag checks if a specific flag is set.
func (h *FECHeader) HasFlag(flag uint16) bool {
	return h.Flags&flag != 0
}

// IsSYN returns true if this is a SYN packet.
func (h *FECHeader) IsSYN() bool {
	return h.HasFlag(FlagSYN)
}

// IsFIN returns true if this is a FIN packet.
func (h *FECHeader) IsFIN() bool {
	return h.HasFlag(FlagFIN)
}

// IsACK returns true if this packet contains an acknowledgment.
func (h *FECHeader) IsACK() bool {
	return h.HasFlag(FlagACK)
}

// IsData returns true if this packet contains data.
func (h *FECHeader) IsData() bool {
	return h.HasFlag(FlagDAT)
}

// SynData represents the RDPUDP_SYNDATA_PAYLOAD structure.
// [MS-RDPEUDP] Section 2.2.2.1
type SynData struct {
	SnInitialSequenceNumber uint32 // Initial sequence number
	UpstreamMTU             uint16 // Client-to-server MTU
	DownstreamMTU           uint16 // Server-to-client MTU
}

// Size returns the serialized size of SynData.
func (s *SynData) Size() int {
	return SynDataSize
}

// Deserialize parses SynData from bytes.
func (s *SynData) Deserialize(data []byte) error {
	if len(data) < SynDataSize {
		return fmt.Errorf("%w: SYN data too short", ErrInvalidPacket)
	}

	s.SnInitialSequenceNumber = binary.LittleEndian.Uint32(data[0:4])
	s.UpstreamMTU = binary.LittleEndian.Uint16(data[4:6])
	s.DownstreamMTU = binary.LittleEndian.Uint16(data[6:8])
	return nil
}

// Serialize encodes SynData to bytes.
func (s *SynData) Serialize() []byte {
	buf := make([]byte, SynDataSize)
	binary.LittleEndian.PutUint32(buf[0:4], s.SnInitialSequenceNumber)
	binary.LittleEndian.PutUint16(buf[4:6], s.UpstreamMTU)
	binary.LittleEndian.PutUint16(buf[6:8], s.DownstreamMTU)
	return buf
}

// AckVector represents the RDPUDP_ACK_VECTOR_HEADER structure.
// [MS-RDPEUDP] Section 2.2.1.2
type AckVector struct {
	AckVectorSize     uint8   // Number of bytes in the ack vector
	Reserved          uint8   // Reserved, must be 0
	AckVectorElements []uint8 // Each bit represents a packet state
}

// Size returns the serialized size of AckVector.
func (a *AckVector) Size() int {
	return 2 + len(a.AckVectorElements)
}

// Deserialize parses AckVector from bytes.
func (a *AckVector) Deserialize(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("%w: ACK vector header too short", ErrInvalidPacket)
	}

	a.AckVectorSize = data[0]
	a.Reserved = data[1]

	if int(a.AckVectorSize) > len(data)-2 {
		return fmt.Errorf("%w: ACK vector elements truncated", ErrInvalidPacket)
	}

	a.AckVectorElements = make([]uint8, a.AckVectorSize)
	copy(a.AckVectorElements, data[2:2+a.AckVectorSize])
	return nil
}

// Serialize encodes AckVector to bytes.
func (a *AckVector) Serialize() []byte {
	buf := make([]byte, a.Size())
	buf[0] = a.AckVectorSize
	buf[1] = a.Reserved
	copy(buf[2:], a.AckVectorElements)
	return buf
}

// SourcePayloadHeader represents the RDPUDP_SOURCE_PAYLOAD_HEADER structure.
// [MS-RDPEUDP] Section 2.2.3.1
type SourcePayloadHeader struct {
	SnSourceStart uint32 // Starting sequence number of this data
	SnCoded       uint32 // Present only if FEC flag is set
	FECMode       uint8  // Present only if FEC flag is set
}

// Size returns the serialized size depending on FEC mode.
func (h *SourcePayloadHeader) Size(hasFEC bool) int {
	if hasFEC {
		return 9 // snSourceStart(4) + snCoded(4) + fecMode(1)
	}
	return 4 // snSourceStart(4) only
}

// Deserialize parses SourcePayloadHeader from bytes.
func (h *SourcePayloadHeader) Deserialize(data []byte, hasFEC bool) error {
	minSize := 4
	if hasFEC {
		minSize = 9
	}
	if len(data) < minSize {
		return fmt.Errorf("%w: source payload header too short", ErrInvalidPacket)
	}

	h.SnSourceStart = binary.LittleEndian.Uint32(data[0:4])
	if hasFEC {
		h.SnCoded = binary.LittleEndian.Uint32(data[4:8])
		h.FECMode = data[8]
	}
	return nil
}

// Serialize encodes SourcePayloadHeader to bytes.
func (h *SourcePayloadHeader) Serialize(hasFEC bool) []byte {
	size := h.Size(hasFEC)
	buf := make([]byte, size)
	binary.LittleEndian.PutUint32(buf[0:4], h.SnSourceStart)
	if hasFEC {
		binary.LittleEndian.PutUint32(buf[4:8], h.SnCoded)
		buf[8] = h.FECMode
	}
	return buf
}

// Packet represents a complete RDPEUDP packet.
type Packet struct {
	Header      FECHeader
	SynData     *SynData
	AckVector   *AckVector
	DataHeader  *SourcePayloadHeader
	Payload     []byte
}

// Deserialize parses a complete RDPEUDP packet.
func (p *Packet) Deserialize(data []byte) error {
	if err := p.Header.Deserialize(data); err != nil {
		return err
	}

	offset := p.Header.Size()

	// Parse SYN data if present
	if p.Header.HasFlag(FlagSYN) {
		p.SynData = &SynData{}
		if err := p.SynData.Deserialize(data[offset:]); err != nil {
			return err
		}
		offset += p.SynData.Size()
	}

	// Parse ACK vector if present
	if p.Header.HasFlag(FlagACK) && p.Header.HasFlag(FlagACKV) {
		p.AckVector = &AckVector{}
		if err := p.AckVector.Deserialize(data[offset:]); err != nil {
			return err
		}
		offset += p.AckVector.Size()
	}

	// Parse data header and payload if present
	if p.Header.HasFlag(FlagDAT) {
		p.DataHeader = &SourcePayloadHeader{}
		hasFEC := p.Header.HasFlag(FlagFEC)
		if err := p.DataHeader.Deserialize(data[offset:], hasFEC); err != nil {
			return err
		}
		offset += p.DataHeader.Size(hasFEC)

		// Remaining bytes are payload
		if offset < len(data) {
			p.Payload = make([]byte, len(data)-offset)
			copy(p.Payload, data[offset:])
		}
	}

	return nil
}

// Serialize encodes a complete RDPEUDP packet.
func (p *Packet) Serialize() []byte {
	var buf bytes.Buffer

	// Write FEC header
	buf.Write(p.Header.Serialize())

	// Write SYN data if present
	if p.SynData != nil && p.Header.HasFlag(FlagSYN) {
		buf.Write(p.SynData.Serialize())
	}

	// Write ACK vector if present
	if p.AckVector != nil && p.Header.HasFlag(FlagACK) && p.Header.HasFlag(FlagACKV) {
		buf.Write(p.AckVector.Serialize())
	}

	// Write data header and payload if present
	if p.DataHeader != nil && p.Header.HasFlag(FlagDAT) {
		hasFEC := p.Header.HasFlag(FlagFEC)
		buf.Write(p.DataHeader.Serialize(hasFEC))
		if len(p.Payload) > 0 {
			buf.Write(p.Payload)
		}
	}

	return buf.Bytes()
}

// NewSYNPacket creates a SYN packet for connection initiation.
func NewSYNPacket(initialSeq uint32, upstreamMTU, downstreamMTU uint16) *Packet {
	return &Packet{
		Header: FECHeader{
			SnSourceAck:              DefaultSnSourceAck,
			SourceAckReceiveWindowSize: DefaultReceiveWindow,
			Flags:                    FlagSYN,
		},
		SynData: &SynData{
			SnInitialSequenceNumber: initialSeq,
			UpstreamMTU:             upstreamMTU,
			DownstreamMTU:           downstreamMTU,
		},
	}
}

// NewSYNACKPacket creates a SYN+ACK packet for connection acceptance.
func NewSYNACKPacket(initialSeq uint32, ackSeq uint32, upstreamMTU, downstreamMTU uint16) *Packet {
	return &Packet{
		Header: FECHeader{
			SnSourceAck:              ackSeq,
			SourceAckReceiveWindowSize: DefaultReceiveWindow,
			Flags:                    FlagSYN | FlagACK,
		},
		SynData: &SynData{
			SnInitialSequenceNumber: initialSeq,
			UpstreamMTU:             upstreamMTU,
			DownstreamMTU:           downstreamMTU,
		},
	}
}

// NewACKPacket creates an ACK-only packet.
func NewACKPacket(ackSeq uint32, receiveWindow uint16) *Packet {
	return &Packet{
		Header: FECHeader{
			SnSourceAck:              ackSeq,
			SourceAckReceiveWindowSize: receiveWindow,
			Flags:                    FlagACK,
		},
	}
}

// NewDataPacket creates a data packet.
func NewDataPacket(seq uint32, ackSeq uint32, data []byte, receiveWindow uint16) *Packet {
	return &Packet{
		Header: FECHeader{
			SnSourceAck:              ackSeq,
			SourceAckReceiveWindowSize: receiveWindow,
			Flags:                    FlagDAT | FlagACK,
		},
		DataHeader: &SourcePayloadHeader{
			SnSourceStart: seq,
		},
		Payload: data,
	}
}

// NewFINPacket creates a FIN packet for connection termination.
func NewFINPacket(ackSeq uint32) *Packet {
	return &Packet{
		Header: FECHeader{
			SnSourceAck:              ackSeq,
			SourceAckReceiveWindowSize: 0,
			Flags:                    FlagFIN | FlagACK,
		},
	}
}

// FlagsString returns a human-readable description of packet flags.
func FlagsString(flags uint16) string {
	var parts []string
	if flags&FlagSYN != 0 {
		parts = append(parts, "SYN")
	}
	if flags&FlagFIN != 0 {
		parts = append(parts, "FIN")
	}
	if flags&FlagACK != 0 {
		parts = append(parts, "ACK")
	}
	if flags&FlagDAT != 0 {
		parts = append(parts, "DAT")
	}
	if flags&FlagFEC != 0 {
		parts = append(parts, "FEC")
	}
	if flags&FlagCN != 0 {
		parts = append(parts, "CN")
	}
	if flags&FlagCWR != 0 {
		parts = append(parts, "CWR")
	}
	if flags&FlagAOA != 0 {
		parts = append(parts, "AOA")
	}
	if flags&FlagSYN2 != 0 {
		parts = append(parts, "SYN2")
	}
	if flags&FlagACKV != 0 {
		parts = append(parts, "ACKV")
	}
	if len(parts) == 0 {
		return "NONE"
	}
	return fmt.Sprintf("%v", parts)
}
