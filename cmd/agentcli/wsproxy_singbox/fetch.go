package wsproxy_singbox

import (
	"encoding/json"
	"fmt"

	"github.com/xhd2015/ai-critic/client"
)

func init() {
	currentHooks.FetchVMess = fetchVMessFromAPI
}

func fetchVMessFromAPI(c *client.Client) (*VMessParams, error) {
	req, err := c.NewRequest("GET", "/api/ws-proxy/vmess-link", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var params VMessParams
	if err := json.NewDecoder(resp.Body).Decode(&params); err != nil {
		return nil, fmt.Errorf("parse vmess response: %w", err)
	}
	return &params, nil
}
