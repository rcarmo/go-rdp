package rdp

import (
	"fmt"
	"io"
	"time"

	"github.com/rcarmo/rdp-html5/internal/logging"
	"github.com/rcarmo/rdp-html5/internal/protocol/pdu"
)

// Connect performs the RDP connection sequence including negotiation,
// TLS/NLA setup, licensing, and capabilities exchange.
func (c *Client) Connect() error {
	var err error

	if err = c.connectionInitiation(); err != nil {
		return fmt.Errorf("connection initiation: %w", err)
	}

	if err = c.basicSettingsExchange(); err != nil {
		return fmt.Errorf("basic settings exchange: %w", err)
	}

	if err = c.channelConnection(); err != nil {
		return fmt.Errorf("channel connection: %w", err)
	}

	if err = c.secureSettingsExchange(); err != nil {
		return fmt.Errorf("secure settings exchange: %w", err)
	}

	if err = c.licensing(); err != nil {
		return fmt.Errorf("licensing: %w", err)
	}

	if err = c.capabilitiesExchange(); err != nil {
		return fmt.Errorf("capabilities exchange: %w", err)
	}

	if err = c.connectionFinalization(); err != nil {
		return fmt.Errorf("connection finalizatioin: %w", err)
	}

	// Request a full screen refresh from the server
	if err = c.sendRefreshRect(); err != nil {
		logging.Warn("RDP: Failed to send refresh rect: %v", err)
		// Don't fail the connection if refresh rect fails
	}

	return nil
}

func (c *Client) connectionInitiation() error {
	var err error

	// Request both SSL and Hybrid (NLA) protocols - server will pick what it supports
	// If useNLA is set, we prefer NLA but will fall back to SSL
	requestedProtocol := c.selectedProtocol
	if c.useNLA {
		// Request both SSL and Hybrid so server can choose
		requestedProtocol = pdu.NegotiationProtocolSSL | pdu.NegotiationProtocolHybrid
	}

	req := pdu.ClientConnectionRequest{
		NegotiationRequest: pdu.NegotiationRequest{
			RequestedProtocols: requestedProtocol,
		},
	}

	var (
		resp pdu.ServerConnectionConfirm
		wire io.Reader
	)

	if wire, err = c.x224Layer.Connect(req.Serialize()); err != nil {
		return err
	}

	if err = resp.Deserialize(wire); err != nil {
		return err
	}

	if resp.Type.IsFailure() {
		failureCode := resp.FailureCode()

		// Provide helpful error messages
		switch failureCode {
		case pdu.NegotiationFailureCodeHybridRequired:
			return fmt.Errorf("server requires Network Level Authentication (NLA/CredSSP). Enable NLA in config or set USE_NLA=true environment variable")
		case pdu.NegotiationFailureCodeSSLRequired:
			return fmt.Errorf("server requires SSL/TLS but negotiation failed")
		case pdu.NegotiationFailureCodeSSLWithUserAuthRequired:
			return fmt.Errorf("server requires SSL with user authentication")
		default:
			return fmt.Errorf("negotiation failure: %s (code=%d)", failureCode.String(), uint32(failureCode))
		}
	}

	c.serverNegotiationFlags = resp.Flags

	selectedProto := resp.SelectedProtocol()

	// Update selectedProtocol to what the server actually chose
	c.selectedProtocol = selectedProto

	// Handle Hybrid (NLA) protocol - preferred when available
	if selectedProto.IsHybrid() {
		return c.StartNLA()
	}

	// Handle SSL protocol
	if selectedProto.IsSSL() {
		return c.StartTLS()
	}

	// Handle standard RDP (no encryption/basic security)
	if selectedProto.IsRDP() {
		// Server selected standard RDP - NLA not supported by server
		return nil
	}
	return ErrUnsupportedRequestedProtocol
}

func (c *Client) basicSettingsExchange() error {
	clientUserDataSet := pdu.NewClientUserDataSet(uint32(c.selectedProtocol), c.desktopWidth, c.desktopHeight, c.colorDepth, c.channels)

	wire, err := c.mcsLayer.Connect(clientUserDataSet.Serialize())
	if err != nil {
		return err
	}

	var serverUserData pdu.ServerUserData
	err = serverUserData.Deserialize(wire)
	if err != nil {
		return err
	}

	c.initChannels(serverUserData.ServerNetworkData)

	// RNS_UD_SC_SKIP_CHANNELJOIN_SUPPORTED = 0x00000008
	// This flag means the server SUPPORTS skipping, but we should only skip if we also requested it
	// For now, always do channel join for maximum compatibility (especially with XRDP)
	// c.skipChannelJoin = serverUserData.ServerCoreData.EarlyCapabilityFlags&0x8 == 0x8
	c.skipChannelJoin = false // Always do channel join for compatibility

	return nil
}

func (c *Client) initChannels(serverNetworkData *pdu.ServerNetworkData) {
	if c.channelIDMap == nil {
		c.channelIDMap = make(map[string]uint16)
	}

	for i, channelName := range c.channels {
		if i < len(serverNetworkData.ChannelIdArray) {
			c.channelIDMap[channelName] = serverNetworkData.ChannelIdArray[i]
		}
	}

	c.channelIDMap["global"] = serverNetworkData.MCSChannelId
}

func (c *Client) channelConnection() error {
	err := c.mcsLayer.ErectDomain()
	if err != nil {
		return err
	}

	c.userID, err = c.mcsLayer.AttachUser()
	if err != nil {
		return err
	}

	c.channelIDMap["user"] = c.userID

	if c.skipChannelJoin {
		return nil
	}

	err = c.mcsLayer.JoinChannels(c.userID, c.channelIDMap)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) secureSettingsExchange() error {
	clientInfoPDU := pdu.NewClientInfo(c.domain, c.username, c.password)

	if c.remoteApp != nil {
		clientInfoPDU.InfoPacket.Flags |= pdu.InfoFlagRail
	}

	// Per MS-RDPBCGR 2.2.1.11.1.1: security header MUST NOT be present when Enhanced RDP Security (TLS) is in effect
	useEnhancedSecurity := c.selectedProtocol.IsSSL() || c.selectedProtocol.IsHybrid()
	data := clientInfoPDU.Serialize(useEnhancedSecurity)

	if err := c.mcsLayer.Send(c.userID, c.channelIDMap["global"], data); err != nil {
		return fmt.Errorf("client info: %w", err)
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *Client) licensing() error {
	// Per MS-RDPBCGR, when Enhanced RDP Security (TLS) is in effect, the security header is not present
	useEnhancedSecurity := c.selectedProtocol.IsSSL() || c.selectedProtocol.IsHybrid()

	// Set a read deadline so we don't hang forever
	if c.conn != nil {
		_ = c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		defer func() { _ = c.conn.SetReadDeadline(time.Time{}) }() // Clear deadline
	}

	_, wire, err := c.mcsLayer.Receive()
	if err != nil {
		errStr := err.Error()
		// Check for disconnect ultimatum which often means authentication failed
		if errStr == "disconnect ultimatum" {
			return fmt.Errorf("server disconnected during licensing - possible causes: 1) Invalid credentials, 2) Account locked, 3) NLA required but not negotiated, 4) XRDP session limit reached")
		}
		return fmt.Errorf("licensing receive: %w", err)
	}

	var resp pdu.ServerLicenseError
	if err = resp.Deserialize(wire, useEnhancedSecurity); err != nil {
		return fmt.Errorf("server license error: %w", err)
	}

	if resp.Preamble.MsgType == 0x03 { // NEW_LICENSE
		return nil
	}

	if resp.Preamble.MsgType != 0xFF { // ERROR_ALERT
		return fmt.Errorf("unknown license msg type: 0x%02X", resp.Preamble.MsgType)
	}

	if resp.ValidClientMessage.ErrorCode != 0x00000007 { // STATUS_VALID_CLIENT
		return fmt.Errorf("license error code: 0x%08X (expected STATUS_VALID_CLIENT 0x00000007)", resp.ValidClientMessage.ErrorCode)
	}

	if resp.ValidClientMessage.StateTransition != 0x00000002 { // ST_NO_TRANSITION
		return fmt.Errorf("license state transition: 0x%08X (expected ST_NO_TRANSITION 0x00000002)", resp.ValidClientMessage.StateTransition)
	}

	return nil
}
