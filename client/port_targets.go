package client

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
)

type OriginTarget struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
}

func (t OriginTarget) URL() string {
	return (&url.URL{Scheme: t.Scheme, Host: net.JoinHostPort(t.Host, strconv.Itoa(t.Port))}).String()
}

type TargetCheck struct {
	Target       OriginTarget `json:"target"`
	TCPReachable bool         `json:"tcpReachable"`
	Error        string       `json:"error,omitempty"`
}
type PortMappingRecord struct {
	ID       string       `json:"id"`
	Target   OriginTarget `json:"target"`
	Hostname string       `json:"hostname"`
	Provider string       `json:"provider"`
	Desired  bool         `json:"desired"`
	Auth     PortAuth     `json:"auth,omitempty"`
}
type PortMappingStore struct {
	Version  int                          `json:"version"`
	Mappings map[string]PortMappingRecord `json:"mappings"`
}

func (c *Client) SavedPortMappings() (PortMappingStore, error) {
	var out PortMappingStore
	err := c.getJSON("/api/ports/mappings", &out)
	return out, err
}
func (c *Client) WritePortMapping(update bool, selector string, body any) (*PortForwardInfo, error) {
	path := "/api/ports"
	method := http.MethodPost
	if update {
		method = http.MethodPut
		path += "?id=" + url.QueryEscape(selector)
	}
	var out PortForwardInfo
	if err := c.sendJSON(method, path, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) RemovePortMapping(selector string, forget bool) error {
	if selector == "" {
		return fmt.Errorf("id or port required")
	}
	q := url.Values{"id": {selector}}
	if forget {
		q.Set("forget", "true")
	}
	return c.sendJSON(http.MethodDelete, "/api/ports?"+q.Encode(), nil, nil)
}
func (c *Client) CheckPortTarget(target OriginTarget) (*TargetCheck, error) {
	var out TargetCheck
	err := c.postJSON("/api/ports/target-check", map[string]any{"target": target}, &out)
	return &out, err
}
