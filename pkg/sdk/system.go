package sdk

func (c *Client) RestartDaemon() error {
	return c.post("/system/restart", nil, nil)
}
