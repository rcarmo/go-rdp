package rdp

// Write writes raw bytes to the underlying RDP connection.
func (c *Client) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}
