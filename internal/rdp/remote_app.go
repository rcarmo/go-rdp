package rdp

// SetRemoteApp configures the client for RAIL (Remote Application) mode.
// Note: RAIL is not supported in the HTML5 client.
func (c *Client) SetRemoteApp(app, args, workingDir string) {
	c.remoteApp = &RemoteApp{
		App:        app,
		WorkingDir: workingDir,
		Args:       args,
	}
	c.channels = append(c.channels, "rail")
	c.railState = RailStateUninitialized
}
