// Package drdynvc implements the Dynamic Virtual Channel Protocol (MS-RDPEDYC).
// This protocol allows creation and management of dynamic virtual channels over RDP.
package drdynvc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Static channel name for DRDYNVC
const ChannelName = "drdynvc"

// Command IDs (MS-RDPEDYC 2.2.1)
const (
	CmdCreate       uint8 = 0x01 // CYCNVC_CREATE_REQ / _RSP
	CmdDataFirst    uint8 = 0x02 // DYNVC_DATA_FIRST
	CmdData         uint8 = 0x03 // DYNVC_DATA
	CmdClose        uint8 = 0x04 // DYNVC_CLOSE
	CmdCapability   uint8 = 0x05 // DYNVC_CAPS_VERSION
	CmdDataFirstCmp uint8 = 0x06 // DYNVC_DATA_FIRST_COMPRESSED (v3)
	CmdDataCmp      uint8 = 0x07 // DYNVC_DATA_COMPRESSED (v3)
	CmdSoftSync     uint8 = 0x08 // DYNVC_SOFT_SYNC_REQUEST / _RESPONSE (v3)
)

// Capability versions
const (
	CapsVersion1 uint16 = 0x0001
	CapsVersion2 uint16 = 0x0002
	CapsVersion3 uint16 = 0x0003
)

// Create request/response result codes
const (
	CreateResultOK              uint32 = 0x00000000
	CreateResultDenied          uint32 = 0x00000001
	CreateResultNoMemory        uint32 = 0x00000002
	CreateResultNoListener      uint32 = 0x00000003
	CreateResultChannelNotFound uint32 = 0x80070490
)

// Header represents the common DRDYNVC PDU header
type Header struct {
	CbChID uint8 // Length of ChannelId field (0=1byte, 1=2bytes, 2=4bytes)
	Sp     uint8 // Varies by command type
	Cmd    uint8 // Command identifier
}

// Serialize encodes the header byte
func (h *Header) Serialize() byte {
	return (h.CbChID & 0x03) | ((h.Sp & 0x03) << 2) | ((h.Cmd & 0x0F) << 4)
}

// Deserialize decodes the header byte
func (h *Header) Deserialize(b byte) {
	h.CbChID = b & 0x03
	h.Sp = (b >> 2) & 0x03
	h.Cmd = (b >> 4) & 0x0F
}

// ChannelIDSize returns the byte size for channel ID based on CbChID
func (h *Header) ChannelIDSize() int {
	switch h.CbChID {
	case 0:
		return 1
	case 1:
		return 2
	case 2:
		return 4
	default:
		return 1
	}
}

// CapsPDU represents DYNVC_CAPS (MS-RDPEDYC 2.2.1.1)
type CapsPDU struct {
	Version uint16
	// Version 3 adds:
	PriorityCharge0 uint16 // Charge for priority 0
	PriorityCharge1 uint16 // Charge for priority 1
	PriorityCharge2 uint16 // Charge for priority 2
	PriorityCharge3 uint16 // Charge for priority 3
}

// Serialize encodes CapsPDU to wire format
func (c *CapsPDU) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Header: CbChID=0, Sp=0 (pad), Cmd=CmdCapability
	header := Header{CbChID: 0, Sp: 0, Cmd: CmdCapability}
	buf.WriteByte(header.Serialize())
	buf.WriteByte(0) // Pad

	_ = binary.Write(buf, binary.LittleEndian, c.Version)

	if c.Version >= CapsVersion3 {
		_ = binary.Write(buf, binary.LittleEndian, c.PriorityCharge0)
		_ = binary.Write(buf, binary.LittleEndian, c.PriorityCharge1)
		_ = binary.Write(buf, binary.LittleEndian, c.PriorityCharge2)
		_ = binary.Write(buf, binary.LittleEndian, c.PriorityCharge3)
	}

	return buf.Bytes()
}

// Deserialize decodes CapsPDU from wire format
func (c *CapsPDU) Deserialize(r io.Reader) error {
	var headerByte, pad byte
	if err := binary.Read(r, binary.LittleEndian, &headerByte); err != nil {
		return fmt.Errorf("caps header: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &pad); err != nil {
		return fmt.Errorf("caps pad: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &c.Version); err != nil {
		return fmt.Errorf("caps version: %w", err)
	}

	// Version 3 has priority charges
	if c.Version >= CapsVersion3 {
		_ = binary.Read(r, binary.LittleEndian, &c.PriorityCharge0)
		_ = binary.Read(r, binary.LittleEndian, &c.PriorityCharge1)
		_ = binary.Read(r, binary.LittleEndian, &c.PriorityCharge2)
		_ = binary.Read(r, binary.LittleEndian, &c.PriorityCharge3)
	}

	return nil
}

// CreateRequestPDU represents DYNVC_CREATE_REQ (MS-RDPEDYC 2.2.2.1)
type CreateRequestPDU struct {
	ChannelID   uint32
	ChannelName string
}

// Serialize encodes CreateRequestPDU to wire format
func (c *CreateRequestPDU) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Determine channel ID size
	var cbChID uint8
	switch {
	case c.ChannelID <= 0xFF:
		cbChID = 0
	case c.ChannelID <= 0xFFFF:
		cbChID = 1
	default:
		cbChID = 2
	}

	// Header: Cmd=CmdCreate, Pri=0
	header := Header{CbChID: cbChID, Sp: 0, Cmd: CmdCreate}
	buf.WriteByte(header.Serialize())

	// Channel ID (variable size)
	switch cbChID {
	case 0:
		buf.WriteByte(byte(c.ChannelID))
	case 1:
		_ = binary.Write(buf, binary.LittleEndian, uint16(c.ChannelID))
	case 2:
		_ = binary.Write(buf, binary.LittleEndian, c.ChannelID)
	}

	// Channel name (null-terminated)
	buf.WriteString(c.ChannelName)
	buf.WriteByte(0)

	return buf.Bytes()
}

// CreateResponsePDU represents DYNVC_CREATE_RSP (MS-RDPEDYC 2.2.2.2)
type CreateResponsePDU struct {
	ChannelID    uint32
	CreationCode uint32
}

// Deserialize decodes CreateResponsePDU from wire format
func (c *CreateResponsePDU) Deserialize(r io.Reader, cbChID uint8) error {
	// Read channel ID based on size
	switch cbChID {
	case 0:
		var id uint8
		if err := binary.Read(r, binary.LittleEndian, &id); err != nil {
			return err
		}
		c.ChannelID = uint32(id)
	case 1:
		var id uint16
		if err := binary.Read(r, binary.LittleEndian, &id); err != nil {
			return err
		}
		c.ChannelID = uint32(id)
	case 2:
		if err := binary.Read(r, binary.LittleEndian, &c.ChannelID); err != nil {
			return err
		}
	}

	// Creation code (HRESULT)
	return binary.Read(r, binary.LittleEndian, &c.CreationCode)
}

// IsSuccess returns true if channel creation succeeded
func (c *CreateResponsePDU) IsSuccess() bool {
	return c.CreationCode == CreateResultOK
}

// DataFirstPDU represents DYNVC_DATA_FIRST (MS-RDPEDYC 2.2.3.1)
type DataFirstPDU struct {
	ChannelID uint32
	Length    uint32 // Total uncompressed length
	Data      []byte
}

// Serialize encodes DataFirstPDU to wire format
func (d *DataFirstPDU) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Determine channel ID size
	var cbChID uint8
	switch {
	case d.ChannelID <= 0xFF:
		cbChID = 0
	case d.ChannelID <= 0xFFFF:
		cbChID = 1
	default:
		cbChID = 2
	}

	// Determine length field size (Sp field)
	var lenSize uint8
	switch {
	case d.Length <= 0xFF:
		lenSize = 0
	case d.Length <= 0xFFFF:
		lenSize = 1
	default:
		lenSize = 2
	}

	// Header
	header := Header{CbChID: cbChID, Sp: lenSize, Cmd: CmdDataFirst}
	buf.WriteByte(header.Serialize())

	// Channel ID
	switch cbChID {
	case 0:
		buf.WriteByte(byte(d.ChannelID))
	case 1:
		_ = binary.Write(buf, binary.LittleEndian, uint16(d.ChannelID))
	case 2:
		_ = binary.Write(buf, binary.LittleEndian, d.ChannelID)
	}

	// Length
	switch lenSize {
	case 0:
		buf.WriteByte(byte(d.Length))
	case 1:
		_ = binary.Write(buf, binary.LittleEndian, uint16(d.Length))
	case 2:
		_ = binary.Write(buf, binary.LittleEndian, d.Length)
	}

	// Data
	buf.Write(d.Data)

	return buf.Bytes()
}

// DataPDU represents DYNVC_DATA (MS-RDPEDYC 2.2.3.2)
type DataPDU struct {
	ChannelID uint32
	Data      []byte
}

// Serialize encodes DataPDU to wire format
func (d *DataPDU) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Determine channel ID size
	var cbChID uint8
	switch {
	case d.ChannelID <= 0xFF:
		cbChID = 0
	case d.ChannelID <= 0xFFFF:
		cbChID = 1
	default:
		cbChID = 2
	}

	// Header
	header := Header{CbChID: cbChID, Sp: 0, Cmd: CmdData}
	buf.WriteByte(header.Serialize())

	// Channel ID
	switch cbChID {
	case 0:
		buf.WriteByte(byte(d.ChannelID))
	case 1:
		_ = binary.Write(buf, binary.LittleEndian, uint16(d.ChannelID))
	case 2:
		_ = binary.Write(buf, binary.LittleEndian, d.ChannelID)
	}

	// Data
	buf.Write(d.Data)

	return buf.Bytes()
}

// ClosePDU represents DYNVC_CLOSE (MS-RDPEDYC 2.2.4)
type ClosePDU struct {
	ChannelID uint32
}

// Serialize encodes ClosePDU to wire format
func (c *ClosePDU) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Determine channel ID size
	var cbChID uint8
	switch {
	case c.ChannelID <= 0xFF:
		cbChID = 0
	case c.ChannelID <= 0xFFFF:
		cbChID = 1
	default:
		cbChID = 2
	}

	// Header
	header := Header{CbChID: cbChID, Sp: 0, Cmd: CmdClose}
	buf.WriteByte(header.Serialize())

	// Channel ID
	switch cbChID {
	case 0:
		buf.WriteByte(byte(c.ChannelID))
	case 1:
		_ = binary.Write(buf, binary.LittleEndian, uint16(c.ChannelID))
	case 2:
		_ = binary.Write(buf, binary.LittleEndian, c.ChannelID)
	}

	return buf.Bytes()
}

// ParsePDU parses a DRDYNVC PDU and returns the command type and remaining data
func ParsePDU(data []byte) (cmd uint8, cbChID uint8, remaining []byte, err error) {
	if len(data) < 1 {
		return 0, 0, nil, fmt.Errorf("DRDYNVC PDU too short")
	}

	var header Header
	header.Deserialize(data[0])

	return header.Cmd, header.CbChID, data[1:], nil
}

// ReadChannelID reads a channel ID from data based on cbChID
func ReadChannelID(data []byte, cbChID uint8) (channelID uint32, remaining []byte, err error) {
	size := 1
	switch cbChID {
	case 0:
		size = 1
	case 1:
		size = 2
	case 2:
		size = 4
	}

	if len(data) < size {
		return 0, nil, fmt.Errorf("not enough data for channel ID")
	}

	switch cbChID {
	case 0:
		channelID = uint32(data[0])
	case 1:
		channelID = uint32(binary.LittleEndian.Uint16(data[:2]))
	case 2:
		channelID = binary.LittleEndian.Uint32(data[:4])
	}

	return channelID, data[size:], nil
}

// Soft-Sync flags (MS-RDPEDYC 2.2.5.1)
const (
	SoftSyncTCPFlushed         uint8 = 0x01
	SoftSyncChannelListPresent uint8 = 0x02
)

// Tunnel types for Soft-Sync (MS-RDPEDYC 2.2.5.1.1)
const (
	TunnelTypeUDPFECR uint32 = 0x00000001 // Reliable UDP with FEC
	TunnelTypeUDPFECL uint32 = 0x00000003 // Lossy UDP with FEC
)

// SoftSyncChannelDef represents a channel in the Soft-Sync channel list
type SoftSyncChannelDef struct {
	ChannelID    uint32
	TunnelType   uint32
}

// SoftSyncRequestPDU represents DYNVC_SOFT_SYNC_REQUEST (MS-RDPEDYC 2.2.5.1)
// Sent by server to initiate TCP to UDP transport transition
type SoftSyncRequestPDU struct {
	Pad           uint8
	Flags         uint8
	NumberOfTunnels uint16
	Channels      []SoftSyncChannelDef
}

// Deserialize decodes SoftSyncRequestPDU from wire format (after header byte)
func (s *SoftSyncRequestPDU) Deserialize(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &s.Pad); err != nil {
		return fmt.Errorf("soft-sync pad: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &s.Flags); err != nil {
		return fmt.Errorf("soft-sync flags: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &s.NumberOfTunnels); err != nil {
		return fmt.Errorf("soft-sync tunnel count: %w", err)
	}

	// Read channel list if present
	if s.Flags&SoftSyncChannelListPresent != 0 {
		var channelCount uint16
		if err := binary.Read(r, binary.LittleEndian, &channelCount); err != nil {
			return fmt.Errorf("soft-sync channel count: %w", err)
		}

		if channelCount > 1024 { // Sanity check
			return fmt.Errorf("too many soft-sync channels: %d", channelCount)
		}

		s.Channels = make([]SoftSyncChannelDef, channelCount)
		for i := uint16(0); i < channelCount; i++ {
			var channelID, tunnelType uint32
			if err := binary.Read(r, binary.LittleEndian, &channelID); err != nil {
				return fmt.Errorf("soft-sync channel %d id: %w", i, err)
			}
			if err := binary.Read(r, binary.LittleEndian, &tunnelType); err != nil {
				return fmt.Errorf("soft-sync channel %d tunnel: %w", i, err)
			}
			s.Channels[i] = SoftSyncChannelDef{
				ChannelID:  channelID,
				TunnelType: tunnelType,
			}
		}
	}

	return nil
}

// SoftSyncResponsePDU represents DYNVC_SOFT_SYNC_RESPONSE (MS-RDPEDYC 2.2.5.2)
// Sent by client in response to Soft-Sync request
type SoftSyncResponsePDU struct {
	Pad             uint8
	NumberOfTunnels uint32
	TunnelTypes     []uint32 // One per tunnel
}

// Serialize encodes SoftSyncResponsePDU to wire format
func (s *SoftSyncResponsePDU) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Header: Cmd=CmdSoftSync
	header := Header{CbChID: 0, Sp: 0, Cmd: CmdSoftSync}
	buf.WriteByte(header.Serialize())
	buf.WriteByte(s.Pad)
	_ = binary.Write(buf, binary.LittleEndian, s.NumberOfTunnels)

	for _, tt := range s.TunnelTypes {
		_ = binary.Write(buf, binary.LittleEndian, tt)
	}

	return buf.Bytes()
}

// DataCompressedPDU represents DYNVC_DATA_COMPRESSED or DYNVC_DATA_FIRST_COMPRESSED
// (MS-RDPEDYC 2.2.3.3 and 2.2.3.4)
type DataCompressedPDU struct {
	ChannelID      uint32
	Length         uint32 // Only for DataFirstCompressed
	CompressedData []byte // RDP8 (ZGFX) compressed data
	IsFirst        bool   // True if this is DATA_FIRST_COMPRESSED
}

// Deserialize decodes DataCompressedPDU from wire format
func (d *DataCompressedPDU) Deserialize(data []byte, cbChID uint8, isFirst bool) error {
	d.IsFirst = isFirst

	channelID, remaining, err := ReadChannelID(data, cbChID)
	if err != nil {
		return fmt.Errorf("compressed data channel id: %w", err)
	}
	d.ChannelID = channelID

	if isFirst {
		// DATA_FIRST_COMPRESSED has a length field
		if len(remaining) < 4 {
			return fmt.Errorf("compressed data first: not enough data for length")
		}
		d.Length = binary.LittleEndian.Uint32(remaining[:4])
		remaining = remaining[4:]
	}

	d.CompressedData = remaining
	return nil
}

// Decompress decompresses the RDP8/ZGFX compressed data
// Returns the decompressed data or error
func (d *DataCompressedPDU) Decompress(decompressor *ZGFXDecompressor) ([]byte, error) {
	if decompressor == nil {
		return nil, fmt.Errorf("no ZGFX decompressor available")
	}
	return decompressor.Decompress(d.CompressedData)
}

// ZGFXDecompressor handles RDP8 bulk compression (MS-RDPEGFX 3.3)
// This is a stateful decompressor that maintains history across calls
type ZGFXDecompressor struct {
	history     []byte
	historyIdx  int
}

// NewZGFXDecompressor creates a new ZGFX decompressor
func NewZGFXDecompressor() *ZGFXDecompressor {
	return &ZGFXDecompressor{
		history:    make([]byte, 2500000), // 2.5MB history buffer per MS-RDPEGFX
		historyIdx: 0,
	}
}

// Decompress decompresses ZGFX/RDP8 compressed data
// Based on MS-RDPEGFX 3.3.1.2 - RDP8 Bulk Compression
func (z *ZGFXDecompressor) Decompress(compressed []byte) ([]byte, error) {
	if len(compressed) == 0 {
		return nil, fmt.Errorf("empty compressed data")
	}

	// Check descriptor byte
	descriptor := compressed[0]
	
	// Bit 0: PACKET_COMPRESSED
	isCompressed := (descriptor & 0x01) != 0
	
	if !isCompressed {
		// Uncompressed data follows descriptor
		data := compressed[1:]
		z.updateHistory(data)
		return data, nil
	}

	// ZGFX compressed - use segmented or single segment
	// Bit 1: PACKET_AT_FRONT (history offset 0)
	// Bit 2: PACKET_FLUSHED (reset history)
	
	if (descriptor & 0x04) != 0 {
		// PACKET_FLUSHED - reset history
		z.historyIdx = 0
	}

	// Decompress using LZSS-like algorithm
	return z.decompressSegment(compressed[1:])
}

// decompressSegment handles the actual ZGFX decompression
func (z *ZGFXDecompressor) decompressSegment(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("segment too short")
	}

	// Read segment header
	segmentCount := binary.LittleEndian.Uint16(data[0:2])
	uncompressedSize := binary.LittleEndian.Uint32(append(data[2:4], 0, 0)) // 16-bit in simple case
	
	if segmentCount == 0 {
		// Single segment mode
		return z.decompressSingleSegment(data[4:], int(uncompressedSize))
	}

	// Multi-segment mode
	result := make([]byte, 0, uncompressedSize)
	offset := 4

	for i := uint16(0); i < segmentCount && offset < len(data); i++ {
		if offset+4 > len(data) {
			break
		}
		segSize := binary.LittleEndian.Uint32(data[offset : offset+4])
		offset += 4

		if offset+int(segSize) > len(data) {
			return nil, fmt.Errorf("segment %d overflows data", i)
		}

		segData, err := z.decompressSingleSegment(data[offset:offset+int(segSize)], 65535)
		if err != nil {
			return nil, fmt.Errorf("segment %d: %w", i, err)
		}
		result = append(result, segData...)
		offset += int(segSize)
	}

	return result, nil
}

// decompressSingleSegment decompresses a single ZGFX segment using LZSS
func (z *ZGFXDecompressor) decompressSingleSegment(data []byte, maxSize int) ([]byte, error) {
	result := make([]byte, 0, maxSize)
	reader := newBitReader(data)

	for len(result) < maxSize {
		// Read token type bit
		bit, err := reader.readBit()
		if err != nil {
			break // End of data
		}

		if bit == 0 {
			// Literal byte
			b, err := reader.readBits(8)
			if err != nil {
				break
			}
			result = append(result, byte(b))
			z.appendHistory(byte(b))
		} else {
			// Match reference
			distance, length, err := z.readMatch(reader)
			if err != nil {
				break
			}
			
			// Copy from history
			for i := 0; i < length; i++ {
				idx := z.historyIdx - distance
				if idx < 0 {
					idx += len(z.history)
				}
				b := z.history[idx%len(z.history)]
				result = append(result, b)
				z.appendHistory(b)
			}
		}
	}

	return result, nil
}

// readMatch reads a match reference (distance, length) from the bit stream
func (z *ZGFXDecompressor) readMatch(reader *bitReader) (distance, length int, err error) {
	// Read distance using variable-length encoding
	distBits, err := reader.readBits(1)
	if err != nil {
		return 0, 0, err
	}

	if distBits == 0 {
		// Short distance (1-256)
		d, err := reader.readBits(8)
		if err != nil {
			return 0, 0, err
		}
		distance = int(d) + 1
	} else {
		// Long distance encoding
		prefix, err := reader.readBits(2)
		if err != nil {
			return 0, 0, err
		}
		
		switch prefix {
		case 0: // 9 bits total
			d, err := reader.readBits(8)
			if err != nil {
				return 0, 0, err
			}
			distance = int(d) + 257
		case 1: // 12 bits total  
			d, err := reader.readBits(10)
			if err != nil {
				return 0, 0, err
			}
			distance = int(d) + 513
		case 2: // 16 bits total
			d, err := reader.readBits(14)
			if err != nil {
				return 0, 0, err
			}
			distance = int(d) + 1537
		case 3: // 20+ bits
			d, err := reader.readBits(18)
			if err != nil {
				return 0, 0, err
			}
			distance = int(d) + 17921
		}
	}

	// Read length using variable-length encoding
	lenBits, err := reader.readBits(1)
	if err != nil {
		return 0, 0, err
	}

	if lenBits == 0 {
		// Short length (3-10)
		l, err := reader.readBits(3)
		if err != nil {
			return 0, 0, err
		}
		length = int(l) + 3
	} else {
		// Longer length
		prefix, err := reader.readBits(2)
		if err != nil {
			return 0, 0, err
		}
		
		switch prefix {
		case 0:
			l, err := reader.readBits(4)
			if err != nil {
				return 0, 0, err
			}
			length = int(l) + 11
		case 1:
			l, err := reader.readBits(6)
			if err != nil {
				return 0, 0, err
			}
			length = int(l) + 27
		case 2:
			l, err := reader.readBits(8)
			if err != nil {
				return 0, 0, err
			}
			length = int(l) + 91
		case 3:
			l, err := reader.readBits(14)
			if err != nil {
				return 0, 0, err
			}
			length = int(l) + 347
		}
	}

	return distance, length, nil
}

func (z *ZGFXDecompressor) updateHistory(data []byte) {
	for _, b := range data {
		z.appendHistory(b)
	}
}

func (z *ZGFXDecompressor) appendHistory(b byte) {
	z.history[z.historyIdx%len(z.history)] = b
	z.historyIdx++
}

// bitReader reads individual bits from a byte slice
type bitReader struct {
	data     []byte
	byteIdx  int
	bitIdx   int
}

func newBitReader(data []byte) *bitReader {
	return &bitReader{data: data}
}

func (r *bitReader) readBit() (uint8, error) {
	if r.byteIdx >= len(r.data) {
		return 0, io.EOF
	}

	bit := (r.data[r.byteIdx] >> (7 - r.bitIdx)) & 1
	r.bitIdx++
	if r.bitIdx >= 8 {
		r.bitIdx = 0
		r.byteIdx++
	}
	return bit, nil
}

func (r *bitReader) readBits(n int) (uint32, error) {
	var result uint32
	for i := 0; i < n; i++ {
		bit, err := r.readBit()
		if err != nil {
			return 0, err
		}
		result = (result << 1) | uint32(bit)
	}
	return result, nil
}
