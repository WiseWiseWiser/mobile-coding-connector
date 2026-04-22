package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

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
	var result *RestartServerResult
	err := c.postSSEJSON("/api/server/exec-restart", map[string]any{}, handler, func(ev ServerStreamEvent) error {
		switch ev.Type {
		case "error":
			if ev.Message == "" {
				return fmt.Errorf("server restart failed")
			}
			return errors.New(ev.Message)
		case "done":
			if ev.Success == "false" {
				return errors.New(defaultStreamError(ev.Message, "server restart failed"))
			}
			result = &RestartServerResult{
				Binary:    ev.Binary,
				Directory: ev.Directory,
				Message:   ev.Message,
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
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
