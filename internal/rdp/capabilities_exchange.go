package rdp

import (
	"github.com/rcarmo/go-rdp/internal/protocol/pdu"
)

func (c *Client) capabilitiesExchange() error {
	_, wire, err := c.mcsLayer.Receive()
	if err != nil {
		return err
	}

	var resp pdu.ServerDemandActive
	if err = resp.Deserialize(wire); err != nil {
		return err
	}

	c.shareID = resp.ShareID
	c.serverCapabilitySets = resp.CapabilitySets

	req := pdu.NewClientConfirmActive(resp.ShareID, c.userID, c.desktopWidth, c.desktopHeight, c.remoteApp != nil)

	if c.enableRFX {
		req.CapabilitySets = append(req.CapabilitySets,
			pdu.NewSurfaceCommandsCapabilitySet(),
			pdu.NewBitmapCodecsWithRFXCapabilitySet(),
		)
	}

	return c.mcsLayer.Send(c.userID, c.channelIDMap["global"], req.Serialize())
}
