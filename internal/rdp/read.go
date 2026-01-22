package rdp

// Read reads raw bytes from the RDP connection's buffered reader.
func (c *Client) Read(b []byte) (int, error) {
	return c.buffReader.Read(b)
}
