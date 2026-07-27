package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// LocalPort is one row from GET /api/ports/local.
type LocalPort struct {
	Port    int    `json:"port"`
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Command string `json:"command"`
	Cmdline string `json:"cmdline"`
}

// PortForwardInfo is one persistent port forward from GET /api/ports.
type PortForwardInfo struct {
	LocalPort int    `json:"localPort"`
	Label     string `json:"label"`
	PublicURL string `json:"publicUrl"`
	Status    string `json:"status"`
	Provider  string `json:"provider"`
	Type      string `json:"type"`
	Error     string `json:"error,omitempty"`
}

// PortVisit is an ad-hoc visit session from /api/ports/visit.
type PortVisit struct {
	ID          string  `json:"id"`
	Port        int     `json:"port"`
	ProxyPort   int     `json:"proxy_port,omitempty"`
	PublicURL   string  `json:"public_url"`
	Provider    string  `json:"provider"`
	Hostname    string  `json:"hostname,omitempty"`
	IdleSeconds float64 `json:"idle_seconds"`
	Status      string  `json:"status"`
	Listening   bool    `json:"listening"`
}

// ListLocalPorts returns remote listening ports (GET /api/ports/local).
func (c *Client) ListLocalPorts() ([]LocalPort, error) {
	var out []LocalPort
	if err := c.getJSON("/api/ports/local", &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []LocalPort{}
	}
	return out, nil
}

// ListPortForwards returns persistent port forwards (GET /api/ports).
func (c *Client) ListPortForwards() ([]PortForwardInfo, error) {
	var out []PortForwardInfo
	if err := c.getJSON("/api/ports", &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []PortForwardInfo{}
	}
	return out, nil
}

// StartPortVisit starts an ad-hoc visit (POST /api/ports/visit).
// idle <= 0 means the server default (10m).
func (c *Client) StartPortVisit(port int, provider string, idle time.Duration) (*PortVisit, error) {
	body := map[string]interface{}{
		"port": port,
	}
	if strings.TrimSpace(provider) != "" {
		body["provider"] = provider
	}
	if idle > 0 {
		body["idle_seconds"] = idle.Seconds()
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest(http.MethodPost, "/api/ports/visit", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}
	var out PortVisit
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode /api/ports/visit: %w", err)
	}
	return &out, nil
}

// ListPortVisits lists active ad-hoc visits (GET /api/ports/visit).
func (c *Client) ListPortVisits() ([]PortVisit, error) {
	var out []PortVisit
	if err := c.getJSON("/api/ports/visit", &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []PortVisit{}
	}
	return out, nil
}

// StopPortVisit stops an ad-hoc visit by id or local port string.
func (c *Client) StopPortVisit(idOrPort string) error {
	idOrPort = strings.TrimSpace(idOrPort)
	if idOrPort == "" {
		return fmt.Errorf("id or port required")
	}
	q := url.Values{}
	if _, err := strconv.Atoi(idOrPort); err == nil {
		q.Set("port", idOrPort)
	} else {
		q.Set("id", idOrPort)
	}
	req, err := c.NewRequest(http.MethodDelete, "/api/ports/visit?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
