package rdp

// Close closes the RDP connection and releases resources.
func (c *Client) Close() error {
	if c.remoteApp != nil {
		c.railState = RailStateUninitialized
	}

	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
