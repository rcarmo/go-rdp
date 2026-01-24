package rdp

import "github.com/rcarmo/go-rdp/internal/protocol/fastpath"

// SendInputEvent sends a FastPath input event (mouse, keyboard, etc.) to the server.
func (c *Client) SendInputEvent(data []byte) error {
	return c.fastPath.Send(fastpath.NewInputEventPDU(data))
}
