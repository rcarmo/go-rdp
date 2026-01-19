package rdp

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/rcarmo/rdp-html5/internal/protocol/fastpath"
	"github.com/rcarmo/rdp-html5/internal/protocol/mcs"
	"github.com/rcarmo/rdp-html5/internal/protocol/pdu"
	"github.com/rcarmo/rdp-html5/internal/protocol/tpkt"
	"github.com/rcarmo/rdp-html5/internal/protocol/x224"
)

type RemoteApp struct {
	App        string
	WorkingDir string
	Args       string
}

type Client struct {
	conn       net.Conn
	buffReader *bufio.Reader
	tpktLayer  *tpkt.Protocol
	x224Layer  *x224.Protocol
	mcsLayer   MCSLayer
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
}

const (
	tcpConnectionTimeout = 5 * time.Second
	readBufferSize       = 64 * 1024
)

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

// Known codec GUIDs
var (
	guidNSCodec    = [16]byte{0xB9, 0x1B, 0x8D, 0xCA, 0x0F, 0x00, 0x4F, 0x15, 0x58, 0x9F, 0xAE, 0x2D, 0x1A, 0x87, 0xE2, 0xD6}
	guidRemoteFX   = [16]byte{0x76, 0x77, 0x2F, 0x12, 0xBD, 0x72, 0x44, 0x63, 0xAF, 0xB3, 0xB7, 0x3C, 0x9C, 0x6F, 0x78, 0x86}
	guidImageRemoteFX = [16]byte{0x2C, 0x31, 0xF9, 0x2C, 0x95, 0x78, 0x47, 0x45, 0x80, 0x97, 0x43, 0x60, 0xDF, 0x10, 0x3F, 0x59}
	guidClearCodec = [16]byte{0xE3, 0x1C, 0x97, 0xA6, 0x58, 0x8D, 0x5B, 0x42, 0xAC, 0x18, 0xE0, 0x9B, 0x7D, 0x42, 0xC7, 0xD5}
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
}

// GetServerCapabilities returns a summary of the server's capabilities
func (c *Client) GetServerCapabilities() *ServerCapabilityInfo {
	info := &ServerCapabilityInfo{
		BitmapCodecs: []string{},
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
