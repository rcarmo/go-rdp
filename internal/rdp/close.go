package rdp

func (c *Client) Close() error {
	if c.remoteApp != nil {
		c.railState = RailStateUninitialized
	}

	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
