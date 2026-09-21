package client

import (
	"fmt"

	servergomod "github.com/xhd2015/ai-critic/server/gomod"
)

// GomodAction POSTs a mod-proxy lifecycle action (status/start/stop/...).
func (c *Client) GomodAction(action string, req servergomod.ActionRequest) (*servergomod.ActionResponse, error) {
	if action == "" {
		return nil, fmt.Errorf("gomod action required")
	}
	var out servergomod.ActionResponse
	path := "/api/remote-agent/gomod/" + action
	if err := c.postJSON(path, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
