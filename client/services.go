package client

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type ServicePortForward struct {
	Port       int    `json:"port"`
	Label      string `json:"label,omitempty"`
	Provider   string `json:"provider,omitempty"`
	BaseDomain string `json:"baseDomain,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
}

type ServicePortForwardStatus struct {
	Port       int    `json:"port"`
	Label      string `json:"label,omitempty"`
	Provider   string `json:"provider,omitempty"`
	BaseDomain string `json:"baseDomain,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
	PublicURL  string `json:"publicUrl,omitempty"`
	Status     string `json:"status,omitempty"`
	Error      string `json:"error,omitempty"`
	Active     bool   `json:"active"`
}

type ServiceStatus struct {
	ID             string                    `json:"id"`
	Name           string                    `json:"name"`
	Command        string                    `json:"command"`
	ProjectDir     string                    `json:"projectDir,omitempty"`
	WorkingDir     string                    `json:"workingDir,omitempty"`
	ExtraEnv       map[string]string         `json:"extraEnv,omitempty"`
	EffectivePath  string                    `json:"effectivePath,omitempty"`
	LogPath        string                    `json:"logPath"`
	Status         string                    `json:"status"`
	PID            int                       `json:"pid"`
	LastStartedAt  string                    `json:"lastStartedAt,omitempty"`
	LastExitedAt   string                    `json:"lastExitedAt,omitempty"`
	LastExitError  string                    `json:"lastExitError,omitempty"`
	DesiredRunning bool                      `json:"desiredRunning"`
	PortForward    *ServicePortForwardStatus `json:"portForward,omitempty"`
}

type LogStreamEvent struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
}

func (c *Client) ListServices(projectDir string) ([]ServiceStatus, error) {
	path := "/api/services"
	if strings.TrimSpace(projectDir) != "" {
		path += "?project_dir=" + url.QueryEscape(projectDir)
	}

	var out []ServiceStatus
	if err := c.getJSON(path, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = []ServiceStatus{}
	}
	return out, nil
}

func (c *Client) StartService(id string) (*ServiceStatus, error) {
	req, err := c.NewRequest(http.MethodPost, "/api/services/start?id="+url.QueryEscape(id), nil)
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

	var out ServiceStatus
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode /api/services/start response: %w", err)
	}
	return &out, nil
}

func (c *Client) StopService(id string) error {
	return c.postServiceAction("/api/services/stop", id)
}

func (c *Client) RestartService(id string) error {
	return c.postServiceAction("/api/services/restart", id)
}

func (c *Client) postServiceAction(path string, id string) error {
	req, err := c.NewRequest(http.MethodPost, path+"?id="+url.QueryEscape(id), nil)
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
	return nil
}

func (c *Client) StreamLogFile(path string, lines int, handler func(LogStreamEvent)) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("log path is required")
	}
	if lines <= 0 {
		lines = 100
	}

	req, err := c.NewRequest(http.MethodGet, "/api/logs/stream?path="+url.QueryEscape(path)+"&lines="+url.QueryEscape(fmt.Sprintf("%d", lines)), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var ev LogStreamEvent
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			return fmt.Errorf("decode log stream event: %w", err)
		}
		if handler != nil {
			handler(ev)
		}
		if ev.Type == "error" {
			return errors.New(defaultStreamError(ev.Message, "log stream failed"))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read log stream: %w", err)
	}
	return nil
}
