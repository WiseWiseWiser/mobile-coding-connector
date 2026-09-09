package client

import (
	"fmt"
	"net/http"

	serverqemu "github.com/xhd2015/ai-critic/server/qemu"
)

// QemuGetConfig returns {enabled} from the server qemu.json.
func (c *Client) QemuGetConfig() (*serverqemu.ConfigResponse, error) {
	var out serverqemu.ConfigResponse
	if err := c.sendJSON(http.MethodGet, "/api/remote-agent/qemu/config", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// QemuSetConfig writes {enabled} to the server qemu.json.
func (c *Client) QemuSetConfig(enabled bool) (*serverqemu.ConfigResponse, error) {
	var out serverqemu.ConfigResponse
	body := serverqemu.ConfigResponse{Enabled: enabled}
	if err := c.sendJSON(http.MethodPut, "/api/remote-agent/qemu/config", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// QemuAction POSTs a guest lifecycle action (status/start/stop/...).
func (c *Client) QemuAction(action string, req serverqemu.ActionRequest) (*serverqemu.ActionResponse, error) {
	if action == "" {
		return nil, fmt.Errorf("qemu action required")
	}
	var out serverqemu.ActionResponse
	path := "/api/remote-agent/qemu/" + action
	if err := c.postJSON(path, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// QemuCloudflaredAction POSTs a guest cloudflared action.
func (c *Client) QemuCloudflaredAction(action string, req serverqemu.ActionRequest) (*serverqemu.ActionResponse, error) {
	if action == "" {
		return nil, fmt.Errorf("qemu cloudflared action required")
	}
	var out serverqemu.ActionResponse
	path := "/api/remote-agent/qemu/cloudflared/" + action
	if err := c.postJSON(path, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
