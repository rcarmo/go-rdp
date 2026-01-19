package rdp

func (c *Client) Close() error {
	if c.remoteApp != nil {
		c.railState = RailStateUninitialized
	}

	return c.conn.Close()
}
