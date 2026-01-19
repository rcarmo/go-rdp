package rdp

import (
	"bufio"
	"fmt"
	"net"
	"time"

	"github.com/kulaginds/rdp-html5/internal/pkg/rdp/fastpath"
	"github.com/kulaginds/rdp-html5/internal/pkg/rdp/mcs"
	"github.com/kulaginds/rdp-html5/internal/pkg/rdp/pdu"
	"github.com/kulaginds/rdp-html5/internal/pkg/rdp/tpkt"
	"github.com/kulaginds/rdp-html5/internal/pkg/rdp/x224"
)

type RemoteApp struct {
	App        string
	WorkingDir string
	Args       string
}

type client struct {
	conn       net.Conn
	buffReader *bufio.Reader
	tpktLayer  *tpkt.Protocol
	x224Layer  *x224.Protocol
	mcsLayer   *mcs.Protocol
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
) (*client, error) {
	c := client{
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
func (c *client) SetTLSConfig(skipValidation bool, serverName string) {
	c.skipTLSValidation = skipValidation
	c.tlsServerName = serverName
}

// SetUseNLA enables or disables Network Level Authentication
func (c *client) SetUseNLA(useNLA bool) {
	c.useNLA = useNLA
	if useNLA {
		c.selectedProtocol = pdu.NegotiationProtocolHybrid
	} else {
		c.selectedProtocol = pdu.NegotiationProtocolSSL
	}
}
