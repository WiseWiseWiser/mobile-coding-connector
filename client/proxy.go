package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ProxyServer mirrors server/proxy/proxyconfig.ProxyServer for the
// subset of fields that clients need when listing configured proxies.
// Password is intentionally included because the server's GET endpoint
// returns it verbatim; callers that render proxies to a user should
// mask or omit it.
type ProxyServer struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Protocol string   `json:"protocol,omitempty"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	Domains  []string `json:"domains"`
}

// ListProxies fetches the list of proxy servers configured in the
// server's settings via GET /api/proxy/servers.
func (c *Client) ListProxies() ([]ProxyServer, error) {
	req, err := c.NewRequest(http.MethodGet, "/api/proxy/servers", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}

	var out []ProxyServer
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode proxy servers response: %w", err)
	}
	return out, nil
}
