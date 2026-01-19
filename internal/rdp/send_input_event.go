package rdp

import "github.com/rcarmo/rdp-html5/internal/protocol/fastpath"

func (c *Client) SendInputEvent(data []byte) error {
	return c.fastPath.Send(fastpath.NewInputEventPDU(data))
}
