package client

import "net/http"

// CloudflaredStatus is the origin's configured and effective cloudflared backend.
type CloudflaredStatus struct {
	Backend    string   `json:"backend"`
	Configured string   `json:"configured"`
	Lines      []string `json:"lines"`
	Warning    string   `json:"warning,omitempty"`
	Public     string   `json:"public,omitempty"`
}

// CloudflaredStatus fetches GET /api/remote-agent/cloudflared/status.
func (c *Client) CloudflaredStatus() (*CloudflaredStatus, error) {
	var out CloudflaredStatus
	if err := c.sendJSON(http.MethodGet, "/api/remote-agent/cloudflared/status", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
