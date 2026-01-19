package rdp

func (c *Client) Read(b []byte) (int, error) {
	return c.buffReader.Read(b)
}
