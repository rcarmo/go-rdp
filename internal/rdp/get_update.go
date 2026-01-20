package rdp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/rcarmo/rdp-html5/internal/protocol/audio"
	"github.com/rcarmo/rdp-html5/internal/protocol/pdu"
)

var updateCounter int

// pendingSlowPathUpdate stores a slow-path update converted to fastpath format
var pendingSlowPathUpdate *Update

func (c *Client) GetUpdate() (*Update, error) {
	// If we have a pending slow-path update, return it first
	if pendingSlowPathUpdate != nil {
		update := pendingSlowPathUpdate
		pendingSlowPathUpdate = nil
		return update, nil
	}

	protocol, err := receiveProtocol(c.buffReader)
	if err != nil {
		return nil, err
	}

	updateCounter++

	if protocol.IsX224() {
		update, err := c.getX224Update()
		switch {
		case err == nil:
			if update != nil {
				// Got a converted slow-path bitmap update
				return update, nil
			}
			// Non-bitmap X224 update, try again
			return c.GetUpdate()
		case errors.Is(err, pdu.ErrDeactiateAll):
			return nil, err

		default:
			return nil, fmt.Errorf("get X.224 update: %w", err)
		}
	}

	fpUpdate, err := c.fastPath.Receive()
	if err != nil {
		return nil, err
	}

	// For native FastPath bitmap updates, inject updateType for JS compatibility
	// FastPath data format: [updateHeader (1 byte)] [size (2 bytes)] [data...]
	// JS expects bitmap data to have: [updateType (2 bytes)] [numberRectangles (2 bytes)] [bitmap data...]
	if len(fpUpdate.Data) >= 3 {
		updateCode := fpUpdate.Data[0] & 0x0f
		if updateCode == FastPathUpdateCodeBitmap {
			// Inject updateType (0x0001 for bitmap) after header+size
			oldData := fpUpdate.Data
			newData := make([]byte, len(oldData)+2)
			copy(newData[0:3], oldData[0:3]) // copy header + size
			// Update size field to include the extra 2 bytes
			origSize := binary.LittleEndian.Uint16(oldData[1:3])
			binary.LittleEndian.PutUint16(newData[1:3], origSize+2)
			// Insert updateType
			binary.LittleEndian.PutUint16(newData[3:5], SlowPathUpdateTypeBitmap)
			// Copy rest of data
			copy(newData[5:], oldData[3:])
			fpUpdate.Data = newData
		}
	}

	return &Update{Data: fpUpdate.Data}, nil
}

// Slow-path update types
const (
	SlowPathUpdateTypeOrders      uint16 = 0x0000
	SlowPathUpdateTypeBitmap      uint16 = 0x0001
	SlowPathUpdateTypePalette     uint16 = 0x0002
	SlowPathUpdateTypeSynchronize uint16 = 0x0003
)

// Fastpath update codes (for conversion)
const (
	FastPathUpdateCodeBitmap      uint8 = 0x01
	FastPathUpdateCodePalette     uint8 = 0x02
	FastPathUpdateCodeSynchronize uint8 = 0x03
)

func (c *Client) getX224Update() (*Update, error) {
	channelID, wire, err := c.mcsLayer.Receive()
	if err != nil {
		return nil, err
	}

	if channelID == c.channelIDMap["rail"] {
		err = c.handleRail(wire)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}

	// Handle rdpsnd audio channel
	if channelID == c.channelIDMap[audio.ChannelRDPSND] {
		if c.audioHandler != nil {
			// Read all data from wire
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, wire); err != nil {
				log.Printf("Audio: Error reading channel data: %v", err)
				return nil, nil
			}
			if err := c.audioHandler.HandleChannelData(buf.Bytes()); err != nil {
				log.Printf("Audio: Error handling channel data: %v", err)
			}
		}
		return nil, nil
	}

	// Read ShareControlHeader first to check PDU type
	var shareControlHeader pdu.ShareControlHeader
	if err = shareControlHeader.Deserialize(wire); err != nil {
		return nil, err
	}

	if shareControlHeader.PDUType.IsDeactivateAll() {
		return nil, pdu.ErrDeactiateAll
	}

	// Read ShareDataHeader fields
	var shareID uint32
	var padding uint8
	var streamID uint8
	var uncompressedLength uint16
	var pduType2 pdu.Type2
	var compressedType uint8
	var compressedLength uint16

	binary.Read(wire, binary.LittleEndian, &shareID)
	binary.Read(wire, binary.LittleEndian, &padding)
	binary.Read(wire, binary.LittleEndian, &streamID)
	binary.Read(wire, binary.LittleEndian, &uncompressedLength)
	binary.Read(wire, binary.LittleEndian, &pduType2)
	binary.Read(wire, binary.LittleEndian, &compressedType)
	binary.Read(wire, binary.LittleEndian, &compressedLength)

	// Handle bitmap updates (PDUTYPE2_UPDATE = 0x02)
	if pduType2.IsUpdate() {
		return c.handleSlowPathGraphicsUpdate(wire)
	}

	// Handle error info
	if pduType2.IsErrorInfo() {
		var errorInfo pdu.ErrorInfoPDUData
		if err := errorInfo.Deserialize(wire); err == nil {
			log.Printf("received error info: %s\n", errorInfo.String())
		}
	}

	return nil, nil
}

func (c *Client) handleSlowPathGraphicsUpdate(wire io.Reader) (*Update, error) {
	// Read updateType (2 bytes) - [MS-RDPBCGR] 2.2.9.1.1.3 Slow-Path Graphics Update
	var updateType uint16
	if err := binary.Read(wire, binary.LittleEndian, &updateType); err != nil {
		return nil, err
	}

	// Read the rest of the data (bitmap data including numberRectangles)
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, wire); err != nil && err != io.EOF {
		return nil, err
	}
	updateData := buf.Bytes()

	// Convert to fastpath format for the browser
	// The JavaScript parseBitmapUpdate expects: [updateType (2 bytes)] [numberRectangles (2 bytes)] [bitmap data...]
	// So we need to include the updateType in the data we send

	var fastpathCode uint8
	switch updateType {
	case SlowPathUpdateTypeBitmap:
		fastpathCode = FastPathUpdateCodeBitmap
	case SlowPathUpdateTypePalette:
		fastpathCode = FastPathUpdateCodePalette
	case SlowPathUpdateTypeSynchronize:
		fastpathCode = FastPathUpdateCodeSynchronize
	default:
		// Unknown update type, skip
		return nil, nil
	}

	// Build fastpath-style data for the browser
	// Format: [updateHeader (1 byte)] [size (2 bytes LE)] [updateType (2 bytes LE)] [bitmap data...]
	// The size field should be the size of everything after the updateHeader+size, i.e. updateType + bitmapData
	updateHeader := fastpathCode                 // fragmentation=0 (single), compression=0 (none)
	totalDataSize := uint16(2 + len(updateData)) // updateType (2 bytes) + rest of data

	fpData := make([]byte, 3+2+len(updateData))
	fpData[0] = updateHeader
	binary.LittleEndian.PutUint16(fpData[1:3], totalDataSize)
	binary.LittleEndian.PutUint16(fpData[3:5], updateType)
	copy(fpData[5:], updateData)

	return &Update{Data: fpData}, nil
}
