package rdp

func (c *Client) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}
