package rdp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/rcarmo/go-rdp/internal/protocol/drdynvc"
	"github.com/rcarmo/go-rdp/internal/protocol/rdpedisp"
)

// DisplayControlHandler manages the display control dynamic channel
type DisplayControlHandler struct {
	client           *Client
	drdynvcChannelID uint16          // Static channel ID for drdynvc
	dispChannelID    uint32          // Dynamic channel ID for display control
	caps             *rdpedisp.CapsPDU
	drdynvcVersion   uint16
	ready            bool
	mu               sync.Mutex
	
	// Pending resize request
	pendingWidth  uint32
	pendingHeight uint32
	hasPending    bool
	
	// V3 features
	zgfxDecompressor *drdynvc.ZGFXDecompressor
	softSyncComplete bool
}

// NewDisplayControlHandler creates a new display control handler
func NewDisplayControlHandler(client *Client) *DisplayControlHandler {
	return &DisplayControlHandler{
		client:           client,
		zgfxDecompressor: drdynvc.NewZGFXDecompressor(),
	}
}

// Initialize sets up the display control handler
func (h *DisplayControlHandler) Initialize(drdynvcChannelID uint16) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.drdynvcChannelID = drdynvcChannelID
	h.ready = false
}

// IsReady returns true if the display control channel is established
func (h *DisplayControlHandler) IsReady() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.ready
}

// GetCapabilities returns the server's display control capabilities
func (h *DisplayControlHandler) GetCapabilities() *rdpedisp.CapsPDU {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.caps
}

// HandleDRDYNVC processes DRDYNVC channel data
func (h *DisplayControlHandler) HandleDRDYNVC(data []byte) error {
	if len(data) < 1 {
		return fmt.Errorf("empty DRDYNVC data")
	}

	cmd, cbChID, remaining, err := drdynvc.ParsePDU(data)
	if err != nil {
		return err
	}

	switch cmd {
	case drdynvc.CmdCapability:
		return h.handleCaps(data)
	case drdynvc.CmdCreate:
		return h.handleCreateResponse(cbChID, remaining)
	case drdynvc.CmdData, drdynvc.CmdDataFirst:
		return h.handleData(cbChID, remaining)
	case drdynvc.CmdClose:
		return h.handleClose(cbChID, remaining)
	case drdynvc.CmdDataFirstCmp, drdynvc.CmdDataCmp:
		return h.handleCompressedData(cbChID, remaining, cmd == drdynvc.CmdDataFirstCmp)
	case drdynvc.CmdSoftSync:
		return h.handleSoftSync(remaining)
	default:
		// Ignore unknown commands
		return nil
	}
}

// handleCaps processes DYNVC_CAPS from server
func (h *DisplayControlHandler) handleCaps(data []byte) error {
	caps := &drdynvc.CapsPDU{}
	if err := caps.Deserialize(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("parse DRDYNVC caps: %w", err)
	}

	// xrdp only accepts version 2 or 3 - validate
	if caps.Version < drdynvc.CapsVersion2 {
		return fmt.Errorf("unsupported DRDYNVC version %d (need >= 2)", caps.Version)
	}

	h.mu.Lock()
	h.drdynvcVersion = caps.Version
	h.mu.Unlock()

	// Send caps response - echo back the same version
	response := &drdynvc.CapsPDU{Version: caps.Version}
	return h.sendDRDYNVC(response.Serialize())
}

// handleCreateResponse processes DYNVC_CREATE_RSP from server
func (h *DisplayControlHandler) handleCreateResponse(cbChID uint8, data []byte) error {
	resp := &drdynvc.CreateResponsePDU{}
	if err := resp.Deserialize(bytes.NewReader(data), cbChID); err != nil {
		return fmt.Errorf("parse create response: %w", err)
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if resp.IsSuccess() {
		h.dispChannelID = resp.ChannelID
		// Now we wait for the server to send display control caps
	}

	return nil
}

// handleData processes dynamic channel data
func (h *DisplayControlHandler) handleData(cbChID uint8, data []byte) error {
	channelID, remaining, err := drdynvc.ReadChannelID(data, cbChID)
	if err != nil {
		return err
	}

	h.mu.Lock()
	if channelID != h.dispChannelID {
		h.mu.Unlock()
		return nil // Not our channel
	}
	h.mu.Unlock()

	// Parse display control PDU
	if len(remaining) < 4 {
		return fmt.Errorf("display control data too short")
	}

	pduType, err := rdpedisp.ParsePDUType(remaining)
	if err != nil {
		return err
	}

	if pduType == rdpedisp.PDUTypeCaps {
		caps := &rdpedisp.CapsPDU{}
		if err := caps.Deserialize(bytes.NewReader(remaining)); err != nil {
			return fmt.Errorf("parse display caps: %w", err)
		}

		h.mu.Lock()
		h.caps = caps
		h.ready = true
		
		// If we have a pending resize, send it now
		if h.hasPending {
			width := h.pendingWidth
			height := h.pendingHeight
			h.hasPending = false
			h.mu.Unlock()
			return h.RequestResize(width, height)
		}
		h.mu.Unlock()
	}

	return nil
}

// handleClose processes channel close
func (h *DisplayControlHandler) handleClose(cbChID uint8, data []byte) error {
	channelID, _, err := drdynvc.ReadChannelID(data, cbChID)
	if err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if channelID == h.dispChannelID {
		h.ready = false
		h.dispChannelID = 0
	}

	return nil
}

// RequestDisplayControlChannel requests creation of the display control channel
func (h *DisplayControlHandler) RequestDisplayControlChannel() error {
	h.mu.Lock()
	channelID := uint32(1) // Start with channel ID 1
	h.dispChannelID = channelID
	h.mu.Unlock()

	req := &drdynvc.CreateRequestPDU{
		ChannelID:   channelID,
		ChannelName: rdpedisp.ChannelName,
	}

	return h.sendDRDYNVC(req.Serialize())
}

// RequestResize sends a display resize request to the server
func (h *DisplayControlHandler) RequestResize(width, height uint32) error {
	h.mu.Lock()
	if !h.ready {
		// Queue the request for when the channel is ready
		h.pendingWidth = width
		h.pendingHeight = height
		h.hasPending = true
		h.mu.Unlock()
		return nil
	}

	// Validate against server capabilities
	if h.caps != nil {
		maxArea := h.caps.MaxMonitorAreaSize
		if maxArea > 0 && uint64(width)*uint64(height) > uint64(maxArea) {
			h.mu.Unlock()
			return fmt.Errorf("requested resolution %dx%d exceeds server max area %d", width, height, maxArea)
		}
	}
	
	dispChannelID := h.dispChannelID
	h.mu.Unlock()

	// Build monitor layout PDU
	layoutPDU := rdpedisp.NewSingleMonitorLayout(width, height)
	layoutData := layoutPDU.Serialize()

	// Wrap in DRDYNVC data PDU
	dataPDU := &drdynvc.DataPDU{
		ChannelID: dispChannelID,
		Data:      layoutData,
	}

	return h.sendDRDYNVC(dataPDU.Serialize())
}

// sendDRDYNVC sends data on the DRDYNVC static channel
func (h *DisplayControlHandler) sendDRDYNVC(data []byte) error {
	h.mu.Lock()
	channelID := h.drdynvcChannelID
	client := h.client
	h.mu.Unlock()

	if channelID == 0 {
		return fmt.Errorf("DRDYNVC channel not initialized")
	}

	// Build channel PDU header
	header := &ChannelPDUHeader{
		Flags: ChannelFlagFirst | ChannelFlagLast,
	}
	
	buf := new(bytes.Buffer)
	// Write total length first (4 bytes)
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(data))) // #nosec G115
	// Write flags (4 bytes)
	_ = binary.Write(buf, binary.LittleEndian, uint32(header.Flags))
	buf.Write(data)

	return client.sendChannelData(channelID, buf.Bytes())
}

// sendChannelData sends data on a specific channel
func (c *Client) sendChannelData(channelID uint16, data []byte) error {
	return c.mcsLayer.Send(c.userID, channelID, data)
}

// handleCompressedData processes V3 compressed data PDUs
func (h *DisplayControlHandler) handleCompressedData(cbChID uint8, data []byte, isFirst bool) error {
	compressed := &drdynvc.DataCompressedPDU{}
	if err := compressed.Deserialize(data, cbChID, isFirst); err != nil {
		return fmt.Errorf("parse compressed data: %w", err)
	}

	h.mu.Lock()
	if compressed.ChannelID != h.dispChannelID {
		h.mu.Unlock()
		return nil // Not our channel
	}
	decompressor := h.zgfxDecompressor
	h.mu.Unlock()

	// Decompress the data
	decompressed, err := compressed.Decompress(decompressor)
	if err != nil {
		return fmt.Errorf("decompress data: %w", err)
	}

	// Process as regular data
	return h.processDisplayControlData(decompressed)
}

// handleSoftSync processes V3 Soft-Sync request from server
func (h *DisplayControlHandler) handleSoftSync(data []byte) error {
	req := &drdynvc.SoftSyncRequestPDU{}
	if err := req.Deserialize(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("parse soft-sync request: %w", err)
	}

	// We're TCP-only, so we respond indicating we'll keep using TCP
	// by not specifying any UDP tunnels
	resp := &drdynvc.SoftSyncResponsePDU{
		Pad:             0,
		NumberOfTunnels: 0,
		TunnelTypes:     nil,
	}

	h.mu.Lock()
	h.softSyncComplete = true
	h.mu.Unlock()

	return h.sendDRDYNVC(resp.Serialize())
}

// processDisplayControlData handles decompressed display control PDU data
func (h *DisplayControlHandler) processDisplayControlData(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("display control data too short")
	}

	pduType, err := rdpedisp.ParsePDUType(data)
	if err != nil {
		return err
	}

	if pduType == rdpedisp.PDUTypeCaps {
		caps := &rdpedisp.CapsPDU{}
		if err := caps.Deserialize(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("parse display caps: %w", err)
		}

		h.mu.Lock()
		h.caps = caps
		h.ready = true
		
		if h.hasPending {
			width := h.pendingWidth
			height := h.pendingHeight
			h.hasPending = false
			h.mu.Unlock()
			return h.RequestResize(width, height)
		}
		h.mu.Unlock()
	}

	return nil
}
