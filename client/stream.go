package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// StreamEvent is one decoded SSE JSON payload from a streaming endpoint.
type StreamEvent struct {
	Type         string
	Message      string
	Verbatim     bool
	ID           string
	Layer        string
	Name         string
	Status       string
	Detail       string
	Hint         string
	TryURL       string
	ServerStatus map[string]any
}

// StreamResult is returned by Client.Stream after a successful SSE session.
type StreamResult struct {
	Events []StreamEvent
	Done   map[string]any
}

// Stream consumes an SSE response, collecting events until done or error.
func (c *Client) Stream(method, path string, body any) (*StreamResult, error) {
	var result StreamResult
	err := c.consumeStream(method, path, body, func(ev StreamEvent, raw map[string]any) error {
		result.Events = append(result.Events, ev)
		switch ev.Type {
		case "error":
			msg := ev.Message
			if msg == "" {
				msg = "stream failed"
			}
			return errors.New(msg)
		case "done":
			result.Done = raw
		}
		return nil
	})
	if err != nil {
		return &result, err
	}
	if result.Done == nil {
		return &result, fmt.Errorf("stream ended without completion event")
	}
	return &result, nil
}

// StreamRawBody sends a non-JSON request body for streaming endpoints.
type StreamRawBody struct {
	Reader      io.Reader
	ContentType string
}

func (c *Client) consumeStream(method, path string, body any, onEvent func(StreamEvent, map[string]any) error) error {
	var bodyReader io.Reader
	contentType := ""
	if body != nil {
		if raw, ok := body.(StreamRawBody); ok {
			bodyReader = raw.Reader
			contentType = raw.ContentType
		} else {
			payload, err := json.Marshal(body)
			if err != nil {
				return fmt.Errorf("marshal request: %w", err)
			}
			bodyReader = bytes.NewReader(payload)
			contentType = "application/json"
		}
	}

	req, err := c.NewRequest(method, path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}

	scanner := bufio.NewScanner(resp.Body)
	// Large homes can produce multi-MB SSE data: lines; cap at 32 MiB per frame.
	scanner.Buffer(make([]byte, 256*1024), 32*1024*1024)

	var sawDone bool
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		ev, raw, err := decodeStreamEvent(payload)
		if err != nil {
			return fmt.Errorf("decode stream event: %w", err)
		}
		if onEvent != nil {
			if err := onEvent(ev, raw); err != nil {
				return err
			}
		}
		if ev.Type == "done" {
			sawDone = true
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

func decodeStreamEvent(payload string) (StreamEvent, map[string]any, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return StreamEvent{}, nil, err
	}

	ev := StreamEvent{
		Type:     stringField(raw, "type"),
		Message:  stringField(raw, "message"),
		Verbatim: boolField(raw, "verbatim"),
		ID:       stringField(raw, "id"),
		Layer:    stringField(raw, "layer"),
		Name:     stringField(raw, "name"),
		Status:   stringField(raw, "status"),
		Detail:   stringField(raw, "detail"),
		Hint:     stringField(raw, "hint"),
		TryURL:   stringField(raw, "try_url"),
	}

	switch t := raw["type"]; t {
	case "log":
		ev.Type = "log"
		if ev.Message == "" {
			ev.Message = stringField(raw, "message")
		}
	case "progress":
		ev.Type = "progress"
	case "section":
		ev.Type = "section"
	case "meta":
		ev.Type = "meta"
		if status, ok := raw["server_status"].(map[string]any); ok {
			ev.ServerStatus = status
		}
	case "done":
		ev.Type = "done"
	case "error":
		ev.Type = "error"
	}

	return ev, raw, nil
}

func stringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func boolField(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	b, _ := v.(bool)
	return b
}

// ConsumeStream invokes onEvent for each frame as it arrives, enabling incremental handlers.
func (c *Client) ConsumeStream(method, path string, body any, onEvent func(StreamEvent, map[string]any) error) (map[string]any, error) {
	var done map[string]any
	err := c.consumeStream(method, path, body, func(ev StreamEvent, raw map[string]any) error {
		if ev.Type == "done" {
			done = raw
		}
		if onEvent == nil {
			return nil
		}
		return onEvent(ev, raw)
	})
	return done, err
}