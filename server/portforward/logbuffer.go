package portforward

import (
	"io"
	"strings"
	"sync"
)

const maxLogLines = 200

// LogBuffer is a thread-safe circular buffer that stores log lines
// and implements io.Writer so it can be used as a process output target.
type LogBuffer struct {
	mu    sync.Mutex
	lines []string
}

// NewLogBuffer creates a new log buffer
func NewLogBuffer() *LogBuffer {
	return &LogBuffer{
		lines: make([]string, 0, maxLogLines),
	}
}

// Write implements io.Writer. It splits input by newlines and stores each line.
func (b *LogBuffer) Write(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	text := string(p)
	newLines := strings.Split(text, "\n")
	for _, line := range newLines {
		trimmed := strings.TrimRight(line, "\r")
		if trimmed == "" {
			continue
		}
		b.lines = append(b.lines, trimmed)
		// Keep only last maxLogLines
		if len(b.lines) > maxLogLines {
			b.lines = b.lines[len(b.lines)-maxLogLines:]
		}
	}
	return len(p), nil
}

// Lines returns a copy of all stored log lines
func (b *LogBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]string, len(b.lines))
	copy(result, b.lines)
	return result
}

// Ensure LogBuffer implements io.Writer
var _ io.Writer = (*LogBuffer)(nil)
