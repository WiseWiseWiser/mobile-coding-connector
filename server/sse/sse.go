// Package sse provides utilities for streaming command output via Server-Sent Events.
package sse

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
)

// Writer wraps an http.ResponseWriter for SSE streaming.
type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewWriter initializes SSE headers and returns a Writer.
// Returns nil if the ResponseWriter does not support flushing.
func NewWriter(w http.ResponseWriter) *Writer {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	return &Writer{w: w, flusher: flusher}
}

// Send writes a single SSE data frame (JSON-encoded).
func (s *Writer) Send(data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(s.w, "data: %s\n\n", jsonData)
	s.flusher.Flush()
}

// SendLog sends a log line.
func (s *Writer) SendLog(message string) {
	s.Send(map[string]string{"type": "log", "message": message})
}

// SendError sends an error event.
func (s *Writer) SendError(message string) {
	s.Send(map[string]string{"type": "error", "message": message})
}

// SendDone sends a done event with an optional payload merged in.
func (s *Writer) SendDone(extra map[string]string) {
	data := map[string]string{"type": "done"}
	for k, v := range extra {
		data[k] = v
	}
	s.Send(data)
}

// SendStatus sends a status update event.
func (s *Writer) SendStatus(status string, extra map[string]string) {
	data := map[string]string{"type": "status", "status": status}
	for k, v := range extra {
		data[k] = v
	}
	s.Send(data)
}

// StreamCmd starts cmd, streams its combined stdout/stderr output as SSE log
// events, and returns the command's exit error (nil on success).
// The caller should call SendError/SendDone afterwards based on the returned error.
func (s *Writer) StreamCmd(cmd *exec.Cmd) error {
	return s.StreamCmdFunc(cmd, nil)
}

// StreamCmdFunc is like StreamCmd but calls onLine for each output line before
// sending it as a log event. If onLine returns true, the line is sent as log;
// if false, the line is skipped (the callback should send its own events).
func (s *Writer) StreamCmdFunc(cmd *exec.Cmd, onLine func(line string) bool) error {
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return fmt.Errorf("failed to start: %v", err)
	}

	// Wait for command in background, close pipe when done
	waitErr := make(chan error, 1)
	go func() {
		waitErr <- cmd.Wait()
		pw.Close()
	}()

	// Stream output lines — handles \r for progress output (e.g. git clone)
	scanner := bufio.NewScanner(pr)
	scanner.Split(splitLines)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		// If callback says to send, or no callback set, send as log
		if onLine == nil || onLine(line) {
			s.SendLog(line)
		}
	}
	pr.Close()

	return <-waitErr
}

// splitLines splits on \n or \r, useful for commands that use \r for progress.
func splitLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	for i, b := range data {
		if b == '\n' || b == '\r' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
