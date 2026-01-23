// Package rdp implements a Remote Desktop Protocol client supporting RDP 5+
// with NLA authentication, bitmap updates, and virtual channels.
package rdp

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rcarmo/rdp-html5/internal/protocol/fastpath"
	"github.com/rcarmo/rdp-html5/internal/protocol/mcs"
	"github.com/rcarmo/rdp-html5/internal/protocol/pdu"
	"github.com/rcarmo/rdp-html5/internal/protocol/tpkt"
	"github.com/rcarmo/rdp-html5/internal/protocol/x224"
)

// RemoteApp contains configuration for running a remote application (RAIL).
// Note: RAIL is not supported in the HTML5 client as browsers cannot create
// native OS windows.
type RemoteApp struct {
	App        string
	WorkingDir string
	Args       string
}

// Client represents an RDP client connection to a remote desktop server.
type Client struct {
	mu sync.RWMutex

	conn       net.Conn
	buffReader *bufio.Reader
	tpktLayer  *tpkt.Protocol
	x224Layer  *x224.Protocol
	mcsLayer   mcs.MCSLayer
	fastPath   *fastpath.Protocol

	domain   string
	username string
	password string

	desktopWidth, desktopHeight uint16
	colorDepth                  int

	serverCapabilitySets []pdu.CapabilitySet
	remoteApp            *RemoteApp
	railState            RailState

	selectedProtocol       pdu.NegotiationProtocol
	serverNegotiationFlags pdu.NegotiationResponseFlag
	channels               []string
	channelIDMap           map[string]uint16
	skipChannelJoin        bool
	shareID                uint32
	userID                 uint16

	// TLS configuration
	skipTLSValidation bool
	tlsServerName     string

	// NLA configuration
	useNLA bool

	// Audio handler
	audioHandler *AudioHandler

	// Display control handler for dynamic resize
	displayControl *DisplayControlHandler

	// Multitransport handler for UDP negotiation
	multitransport *MultitransportHandler

	// Pending slow-path update (per-client, not global)
	pendingSlowPathUpdate *Update
}

const (
	tcpConnectionTimeout = 5 * time.Second
	readBufferSize       = 64 * 1024
)

// NewClient creates a new RDP client and establishes a TCP connection to the server.
func NewClient(
	hostname, username, password string,
	desktopWidth, desktopHeight int,
	colorDepth int,
) (*Client, error) {
	// Add default RDP port if not specified
	if !strings.Contains(hostname, ":") {
		hostname = hostname + ":3389"
	}

	c := Client{
		domain:   "",
		username: username,
		password: password,

		desktopWidth:  uint16(desktopWidth),
		desktopHeight: uint16(desktopHeight),
		colorDepth:    colorDepth,

		selectedProtocol: pdu.NegotiationProtocolSSL,
		// Default TLS configuration - can be overridden with SetTLSConfig
		skipTLSValidation: false,
		tlsServerName:     "",
	}

	var err error

	c.conn, err = net.DialTimeout("tcp", hostname, tcpConnectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("tcp connect: %w", err)
	}

	c.buffReader = bufio.NewReaderSize(c.conn, readBufferSize)

	c.tpktLayer = tpkt.New(&c)
	c.x224Layer = x224.New(c.tpktLayer)
	c.mcsLayer = mcs.New(c.x224Layer)
	c.fastPath = fastpath.New(&c)

	return &c, nil
}

// SetTLSConfig allows setting TLS configuration for the RDP client
func (c *Client) SetTLSConfig(skipValidation bool, serverName string) {
	c.skipTLSValidation = skipValidation
	c.tlsServerName = serverName
}

// SetUseNLA enables or disables Network Level Authentication
func (c *Client) SetUseNLA(useNLA bool) {
	c.useNLA = useNLA
	if useNLA {
		c.selectedProtocol = pdu.NegotiationProtocolHybrid
	} else {
		c.selectedProtocol = pdu.NegotiationProtocolSSL
	}
}

// Known codec GUIDs (stored in wire format per MS-RDPBCGR)
// GUID Data1 is 32-bit LE, Data2 is 16-bit LE, Data3 is 16-bit LE, Data4 is 8 bytes BE
var (
	// NSCodec: CA8D1BB9-000F-154F-589F-AE2D1A87E2D6
	guidNSCodec = [16]byte{0xB9, 0x1B, 0x8D, 0xCA, 0x0F, 0x00, 0x4F, 0x15, 0x58, 0x9F, 0xAE, 0x2D, 0x1A, 0x87, 0xE2, 0xD6}
	// RemoteFX: 76772F12-BD72-4463-AFB3-B73C9C6F7886
	guidRemoteFX = [16]byte{0x12, 0x2F, 0x77, 0x76, 0x72, 0xBD, 0x63, 0x44, 0xAF, 0xB3, 0xB7, 0x3C, 0x9C, 0x6F, 0x78, 0x86}
	// RemoteFX Image: 2744CCD4-9D8A-4E74-803C-0ECBEAA19C54
	guidImageRemoteFX = [16]byte{0xD4, 0xCC, 0x44, 0x27, 0x8A, 0x9D, 0x74, 0x4E, 0x80, 0x3C, 0x0E, 0xCB, 0xEA, 0xA1, 0x9C, 0x54}
	// ClearCodec: A6971CE3-8D58-425B-AC18-E09B7D42C7D5
	guidClearCodec = [16]byte{0xE3, 0x1C, 0x97, 0xA6, 0x58, 0x8D, 0x5B, 0x42, 0xAC, 0x18, 0xE0, 0x9B, 0x7D, 0x42, 0xC7, 0xD5}
	// Ignore: 9C4351A6-3535-42AE-910C-CDFCE5760B58
	guidIgnore = [16]byte{0xA6, 0x51, 0x43, 0x9C, 0x35, 0x35, 0xAE, 0x42, 0x91, 0x0C, 0xCD, 0xFC, 0xE5, 0x76, 0x0B, 0x58}
	// RemoteFX Progressive: E329E05D-9B18-4F9D-8EC3-4E4DD1EB3DC1
	guidRemoteFXProgressive = [16]byte{0x5D, 0xE0, 0x29, 0xE3, 0x18, 0x9B, 0x9D, 0x4F, 0x8E, 0xC3, 0x4E, 0x4D, 0xD1, 0xEB, 0x3D, 0xC1}
)

func codecGUIDToName(guid [16]byte) string {
	switch guid {
	case guidNSCodec:
		return "NSCodec"
	case guidRemoteFX:
		return "RemoteFX"
	case guidImageRemoteFX:
		return "RemoteFX-Image"
	case guidClearCodec:
		return "ClearCodec"
	case guidIgnore:
		return "Ignore"
	case guidRemoteFXProgressive:
		return "RemoteFX-Progressive"
	default:
		return fmt.Sprintf("Unknown(%x)", guid[:4])
	}
}

// ServerCapabilityInfo contains a summary of server capabilities for logging
type ServerCapabilityInfo struct {
	BitmapCodecs      []string
	SurfaceCommands   bool
	ColorDepth        int
	DesktopSize       string
	GeneralFlags      uint16
	OrderFlags        uint32
	MultifragmentSize uint32
	LargePointer      bool
	FrameAcknowledge  bool
	// Connection info
	UseNLA       bool
	AudioEnabled bool
	Channels     []string
}

// Update represents an RDP screen update that can be sent to a client.
// This provides a public interface without exposing internal protocol details.
type Update struct {
	Data []byte
}

// GetServerCapabilities returns a summary of the server's capabilities
func (c *Client) GetServerCapabilities() *ServerCapabilityInfo {
	info := &ServerCapabilityInfo{
		BitmapCodecs: []string{},
		UseNLA:       c.useNLA,
		AudioEnabled: c.audioHandler != nil,
		Channels:     c.channels,
	}

	for _, capSet := range c.serverCapabilitySets {
		switch capSet.CapabilitySetType {
		case pdu.CapabilitySetTypeBitmap:
			if capSet.BitmapCapabilitySet != nil {
				info.ColorDepth = int(capSet.BitmapCapabilitySet.PreferredBitsPerPixel)
				info.DesktopSize = fmt.Sprintf("%dx%d", 
					capSet.BitmapCapabilitySet.DesktopWidth, 
					capSet.BitmapCapabilitySet.DesktopHeight)
			}
		case pdu.CapabilitySetTypeGeneral:
			if capSet.GeneralCapabilitySet != nil {
				info.GeneralFlags = capSet.GeneralCapabilitySet.ExtraFlags
			}
		case pdu.CapabilitySetTypeOrder:
			if capSet.OrderCapabilitySet != nil {
				info.OrderFlags = uint32(capSet.OrderCapabilitySet.OrderFlags)
			}
		case pdu.CapabilitySetTypeSurfaceCommands:
			info.SurfaceCommands = true
		case pdu.CapabilitySetTypeBitmapCodecs:
			if capSet.BitmapCodecsCapabilitySet != nil {
				for _, codec := range capSet.BitmapCodecsCapabilitySet.BitmapCodecArray {
					info.BitmapCodecs = append(info.BitmapCodecs, codecGUIDToName(codec.CodecGUID))
				}
			}
		case pdu.CapabilitySetTypeMultifragmentUpdate:
			if capSet.MultifragmentUpdateCapabilitySet != nil {
				info.MultifragmentSize = capSet.MultifragmentUpdateCapabilitySet.MaxRequestSize
			}
		case pdu.CapabilitySetTypeLargePointer:
			info.LargePointer = true
		case pdu.CapabilitySetTypeFrameAcknowledge:
			info.FrameAcknowledge = true
		}
	}

	return info
}

// EnableDisplayControl enables dynamic display resize support.
// This adds the DRDYNVC channel which is required for display control.
func (c *Client) EnableDisplayControl() {
	// Add drdynvc to channels if not already present
	for _, ch := range c.channels {
		if ch == "drdynvc" {
			return
		}
	}
	c.channels = append(c.channels, "drdynvc")
	c.displayControl = NewDisplayControlHandler(c)
}

// IsDisplayControlReady returns true if display control is available
func (c *Client) IsDisplayControlReady() bool {
	if c.displayControl == nil {
		return false
	}
	return c.displayControl.IsReady()
}

// RequestResize requests a display resize via the display control channel.
// Returns an error if display control is not available.
func (c *Client) RequestResize(width, height int) error {
	if c.displayControl == nil {
		return fmt.Errorf("display control not enabled")
	}
	return c.displayControl.RequestResize(uint32(width), uint32(height))
}

// GetDisplayControlCapabilities returns the server's display control capabilities
func (c *Client) GetDisplayControlCapabilities() (maxMonitors uint32, maxArea uint32) {
	if c.displayControl == nil {
		return 0, 0
	}
	caps := c.displayControl.GetCapabilities()
	if caps == nil {
		return 0, 0
	}
	return caps.MaxNumMonitors, caps.MaxMonitorAreaSize
}

// EnableMultitransport enables UDP transport negotiation.
// When enabled, the client will respond to server multitransport requests.
// Note: UDP transport requires additional setup (DTLS, UDP listener).
func (c *Client) EnableMultitransport(enabled bool) {
	if c.multitransport == nil {
		c.multitransport = NewMultitransportHandler(func(data []byte) error {
			return c.sendMultitransportResponse(data)
		})
	}
	c.multitransport.EnableUDP(enabled)
}

// SetMultitransportCallback sets a callback for when UDP transport is ready.
func (c *Client) SetMultitransportCallback(cb func(requestID uint32, cookie [16]byte, reliable bool)) {
	if c.multitransport == nil {
		c.multitransport = NewMultitransportHandler(func(data []byte) error {
			return c.sendMultitransportResponse(data)
		})
	}
	c.multitransport.SetUDPReadyCallback(cb)
}

// HandleMultitransportRequest processes a multitransport request from the server.
func (c *Client) HandleMultitransportRequest(data []byte) error {
	if c.multitransport == nil {
		// Not configured - auto-decline
		c.multitransport = NewMultitransportHandler(func(data []byte) error {
			return c.sendMultitransportResponse(data)
		})
	}
	return c.multitransport.HandleRequest(data)
}

// sendMultitransportResponse sends a multitransport response via the MCS I/O channel.
// Per MS-RDPBCGR Section 2.2.4.17.1, the Initiate Multitransport Response PDU
// is sent as a slow-path data PDU on the I/O channel.
func (c *Client) sendMultitransportResponse(data []byte) error {
	// Get the global (I/O) channel ID
	c.mu.RLock()
	globalChannelID, ok := c.channelIDMap["global"]
	userID := c.userID
	mcsLayer := c.mcsLayer
	c.mu.RUnlock()

	if !ok || globalChannelID == 0 {
		return fmt.Errorf("global channel not established")
	}

	if mcsLayer == nil {
		return fmt.Errorf("MCS layer not initialized")
	}

	// Per MS-RDPBCGR 2.2.4.17.1: The response is sent as SEC_TRANSPORT_RSP
	// For now, we send the raw response data - the MultitransportResponse
	// already contains the proper structure
	if err := mcsLayer.Send(userID, globalChannelID, data); err != nil {
		return fmt.Errorf("send multitransport response: %w", err)
	}

	return nil
}
