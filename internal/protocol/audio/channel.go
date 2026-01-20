// Package audio implements RDP audio virtual channel protocols.
// This file contains virtual channel PDU handling.
package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Channel PDU flags (MS-RDPBCGR 2.2.6.1)
const (
	ChannelFlagFirst    uint32 = 0x00000001
	ChannelFlagLast     uint32 = 0x00000002
	ChannelFlagShowProtocol uint32 = 0x00000010
	ChannelFlagSuspend  uint32 = 0x00000020
	ChannelFlagResume   uint32 = 0x00000040
	ChannelFlagCompress uint32 = 0x00200000
	ChannelFlagPacketAt uint32 = 0x00100000
	ChannelFlagPacketFlushed uint32 = 0x00080000
)

// ChannelPDUHeader represents the virtual channel PDU header
type ChannelPDUHeader struct {
	Length uint32 // Total length of uncompressed channel data
	Flags  uint32 // Channel flags
}

func (h *ChannelPDUHeader) Serialize() []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:4], h.Length)
	binary.LittleEndian.PutUint32(buf[4:8], h.Flags)
	return buf
}

func (h *ChannelPDUHeader) Deserialize(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &h.Length); err != nil {
		return fmt.Errorf("channel header length: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &h.Flags); err != nil {
		return fmt.Errorf("channel header flags: %w", err)
	}
	return nil
}

// IsFirst returns true if this is the first chunk of a fragmented message
func (h *ChannelPDUHeader) IsFirst() bool {
	return h.Flags&ChannelFlagFirst != 0
}

// IsLast returns true if this is the last chunk of a fragmented message
func (h *ChannelPDUHeader) IsLast() bool {
	return h.Flags&ChannelFlagLast != 0
}

// IsComplete returns true if this is a complete (non-fragmented) message
func (h *ChannelPDUHeader) IsComplete() bool {
	return h.IsFirst() && h.IsLast()
}

// ChannelChunk represents a chunk of virtual channel data
type ChannelChunk struct {
	Header ChannelPDUHeader
	Data   []byte
}

// ChannelDefragmenter handles reassembly of fragmented channel PDUs
type ChannelDefragmenter struct {
	buffer    bytes.Buffer
	totalLen  uint32
	receiving bool
}

// Process handles a channel chunk and returns complete data when available
func (d *ChannelDefragmenter) Process(chunk *ChannelChunk) ([]byte, bool) {
	if chunk.Header.IsFirst() {
		d.buffer.Reset()
		d.totalLen = chunk.Header.Length
		d.receiving = true
	}
	
	if !d.receiving {
		return nil, false
	}
	
	d.buffer.Write(chunk.Data)
	
	if chunk.Header.IsLast() {
		d.receiving = false
		return d.buffer.Bytes(), true
	}
	
	return nil, false
}

// ParseChannelData parses raw channel data into header and payload
func ParseChannelData(data []byte) (*ChannelChunk, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("channel data too short: %d bytes", len(data))
	}
	
	chunk := &ChannelChunk{}
	r := bytes.NewReader(data)
	
	if err := chunk.Header.Deserialize(r); err != nil {
		return nil, err
	}
	
	chunk.Data = data[8:]
	return chunk, nil
}

// BuildChannelData creates a channel PDU with the given data
func BuildChannelData(data []byte) []byte {
	header := ChannelPDUHeader{
		Length: uint32(len(data)),
		Flags:  ChannelFlagFirst | ChannelFlagLast,
	}
	
	buf := make([]byte, 8+len(data))
	copy(buf[0:8], header.Serialize())
	copy(buf[8:], data)
	return buf
}

// BuildChannelPDU creates a complete RDPSND PDU
func BuildChannelPDU(msgType uint8, body []byte) []byte {
	header := PDUHeader{
		MsgType:  msgType,
		Reserved: 0,
		BodySize: uint16(len(body)),
	}
	
	pdu := append(header.Serialize(), body...)
	return BuildChannelData(pdu)
}
