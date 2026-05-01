package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type KeepAlivePing struct {
	Running      bool   `json:"running"`
	StartCommand string `json:"start_command,omitempty"`
}

type KeepAliveStatus struct {
	Running             bool   `json:"running"`
	BinaryPath          string `json:"binary_path"`
	DaemonBinaryPath    string `json:"daemon_binary_path"`
	ServerPort          int    `json:"server_port"`
	ServerPID           int    `json:"server_pid"`
	KeepAlivePort       int    `json:"keep_alive_port"`
	KeepAlivePID        int    `json:"keep_alive_pid"`
	StartedAt           string `json:"started_at,omitempty"`
	Uptime              string `json:"uptime,omitempty"`
	NextBinary          string `json:"next_binary,omitempty"`
	NextHealthCheckTime string `json:"next_health_check_time,omitempty"`
	RestartCount        int    `json:"restart_count"`
}

type MemoryStatus struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type DiskStatus struct {
	Filesystem string  `json:"filesystem"`
	Size       uint64  `json:"size"`
	Used       uint64  `json:"used"`
	Available  uint64  `json:"available"`
	UsePercent float64 `json:"use_percent"`
	MountPoint string  `json:"mount_point"`
}

type CPUStatus struct {
	NumCPU      int     `json:"num_cpu"`
	UsedPercent float64 `json:"used_percent"`
}

type OSInfo struct {
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Kernel  string `json:"kernel"`
	Version string `json:"version"`
}

type ProcessStatus struct {
	PID     int    `json:"pid"`
	Name    string `json:"name"`
	CPU     string `json:"cpu"`
	Mem     string `json:"mem"`
	Command string `json:"command"`
}

type ServerStatus struct {
	Memory MemoryStatus    `json:"memory"`
	Disk   []DiskStatus    `json:"disk"`
	CPU    CPUStatus       `json:"cpu"`
	OSInfo OSInfo          `json:"os_info"`
	TopCPU []ProcessStatus `json:"top_cpu"`
	TopMem []ProcessStatus `json:"top_mem"`
}

// ServerStreamEvent is one JSON payload emitted by the server's SSE endpoints.
type ServerStreamEvent struct {
	Type        string `json:"type"`
	Message     string `json:"message,omitempty"`
	Success     string `json:"success,omitempty"`
	Binary      string `json:"binary,omitempty"`
	BinaryPath  string `json:"binary_path,omitempty"`
	BinaryName  string `json:"binary_name,omitempty"`
	Version     string `json:"version,omitempty"`
	Size        string `json:"size,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
	Directory   string `json:"directory,omitempty"`
	Status      string `json:"status,omitempty"`
}

type BuildNextResult struct {
	BinaryPath  string
	BinaryName  string
	Version     string
	Size        string
	ProjectName string
	Message     string
}

type RestartServerResult struct {
	Binary    string
	Directory string
	Message   string
	KeepAlive *KeepAliveStatus
}

func (c *Client) PingKeepAlive() (*KeepAlivePing, error) {
	var out KeepAlivePing
	if err := c.getJSON("/api/keep-alive/ping", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetKeepAliveStatus() (*KeepAliveStatus, error) {
	var out KeepAliveStatus
	if err := c.getJSON("/api/keep-alive/status", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetServerStatus() (*ServerStatus, error) {
	var out ServerStatus
	if err := c.getJSON("/api/server/status", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) BuildNext(projectID string, handler func(ServerStreamEvent)) (*BuildNextResult, error) {
	reqBody := struct {
		ProjectID string `json:"project_id,omitempty"`
	}{
		ProjectID: projectID,
	}

	var result *BuildNextResult
	err := c.postSSEJSON("/api/build/build-next", reqBody, handler, func(ev ServerStreamEvent) error {
		if ev.Type != "done" {
			return nil
		}
		if ev.Success == "false" {
			return errors.New(defaultStreamError(ev.Message, "build-next failed"))
		}
		result = &BuildNextResult{
			BinaryPath:  ev.BinaryPath,
			BinaryName:  ev.BinaryName,
			Version:     ev.Version,
			Size:        ev.Size,
			ProjectName: ev.ProjectName,
			Message:     ev.Message,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) RestartServer(handler func(ServerStreamEvent)) (*RestartServerResult, error) {
	req, err := c.NewRequest(http.MethodPost, "/api/server/exec-restart", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}

	var result *RestartServerResult
	var streamErr error
	var sawDone bool
	var restartTransitionReached bool
	var connectionClosedAfterRestart bool

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		payload := strings.TrimPrefix(line, "data: ")
		var ev ServerStreamEvent
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			return nil, fmt.Errorf("decode stream event: %w", err)
		}
		if handler != nil {
			handler(ev)
		}
		if isRestartTransitionEvent(ev) {
			restartTransitionReached = true
		}

		switch ev.Type {
		case "error":
			if ev.Message == "" {
				streamErr = fmt.Errorf("server restart failed")
				break
			}
			streamErr = errors.New(ev.Message)
		case "done":
			sawDone = true
			if ev.Success == "false" {
				streamErr = errors.New(defaultStreamError(ev.Message, "server restart failed"))
				break
			}
			result = &RestartServerResult{
				Binary:    ev.Binary,
				Directory: ev.Directory,
				Message:   ev.Message,
			}
		}
		if streamErr != nil {
			break
		}
	}

	if streamErr == nil {
		if err := scanner.Err(); err != nil {
			if !restartTransitionReached {
				return nil, fmt.Errorf("read stream: %w", err)
			}
			connectionClosedAfterRestart = true
			emitServerStreamLog(handler, "Connection closed while server restarts; waiting for server to come back...")
		}
	}

	if streamErr != nil {
		return nil, streamErr
	}
	if !sawDone && !restartTransitionReached {
		return nil, fmt.Errorf("stream ended without completion event")
	}
	if result == nil {
		result = &RestartServerResult{
			Message: "Server restart in progress",
		}
	}
	if restartTransitionReached && !connectionClosedAfterRestart {
		emitServerStreamLog(handler, "Waiting for server to come back...")
	}

	if err := c.waitForServerReachable(75*time.Second, 1500*time.Millisecond); err != nil {
		return result, err
	}

	result.KeepAlive = c.getKeepAliveStatusBestEffort()
	return result, nil
}

func isRestartTransitionEvent(ev ServerStreamEvent) bool {
	if ev.Type == "done" && ev.Success != "false" {
		return true
	}
	if ev.Type != "log" {
		return false
	}
	switch ev.Message {
	case "Preparing to exec...",
		"Initiating graceful shutdown (30s max)...",
		"Graceful shutdown completed",
		"Graceful shutdown timeout reached, proceeding with restart":
		return true
	default:
		return false
	}
}

func emitServerStreamLog(handler func(ServerStreamEvent), message string) {
	if handler == nil || strings.TrimSpace(message) == "" {
		return
	}
	handler(ServerStreamEvent{
		Type:    "log",
		Message: message,
	})
}

func (c *Client) waitForServerReachable(timeout time.Duration, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		if err := c.checkAuthWithTimeout(3 * time.Second); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if time.Now().After(deadline) {
			break
		}
		time.Sleep(interval)
	}

	if lastErr == nil {
		return fmt.Errorf("server restart acknowledged, but the server did not become reachable within %v", timeout)
	}
	return fmt.Errorf("server restart acknowledged, but the server did not become reachable within %v: %w", timeout, lastErr)
}

func (c *Client) checkAuthWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := c.NewRequest(http.MethodGet, "/api/auth/check", nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

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

func (c *Client) getKeepAliveStatusBestEffort() *KeepAliveStatus {
	ping, err := c.PingKeepAlive()
	if err != nil || !ping.Running {
		return nil
	}
	status, err := c.GetKeepAliveStatus()
	if err != nil {
		return nil
	}
	return status
}

func (c *Client) postSSEJSON(path string, body any, handler func(ServerStreamEvent), onEvent func(ServerStreamEvent) error) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := c.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
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

	var sawDone bool
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var ev ServerStreamEvent
		if err := json.Unmarshal([]byte(payload), &ev); err != nil {
			return fmt.Errorf("decode stream event: %w", err)
		}
		if handler != nil {
			handler(ev)
		}
		if ev.Type == "done" {
			sawDone = true
		}
		if onEvent != nil {
			if err := onEvent(ev); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}
	if !sawDone {
		return fmt.Errorf("stream ended without completion event")
	}
	return nil
}

func defaultStreamError(message string, fallback string) string {
	if strings.TrimSpace(message) == "" {
		return fallback
	}
	return message
}

func (c *Client) getJSON(path string, out any) error {
	req, err := c.NewRequest(http.MethodGet, path, nil)
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
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s response: %w", path, err)
	}
	return nil
}
