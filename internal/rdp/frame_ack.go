package rdp

import "github.com/rcarmo/go-rdp/internal/protocol/pdu"

// SendFrameAcknowledge sends a Frame Acknowledge PDU to the server
// This is required when using Surface Commands - the server expects
// acknowledgement of received frames
func (c *Client) SendFrameAcknowledge(frameID uint32) error {
	ack := pdu.NewFrameAcknowledgePDU(c.shareID, c.userID, frameID)
	return c.mcsLayer.Send(c.userID, c.channelIDMap["global"], ack.Serialize())
}
