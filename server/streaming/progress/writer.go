package progress

import (
	"net/http"

	"github.com/xhd2015/agent-pro/agent/streaming/sse"
)

// Item is one incremental progress result on the wire.
type Item struct {
	ID     string
	Layer  string
	Name   string
	Status string
	Detail string
	Hint   string
}

// Writer emits typed SSE JSON frames for streaming progress endpoints.
type Writer struct {
	sse *sse.Writer
}

// NewWriter wraps an http.ResponseWriter with SSE headers and flushing.
// Returns nil when w does not implement http.Flusher.
func NewWriter(w http.ResponseWriter) *Writer {
	sw := sse.NewWriter(w)
	if sw == nil {
		return nil
	}
	return &Writer{sse: sw}
}

func (w *Writer) send(data map[string]any) error {
	w.sse.Send(data)
	return nil
}

// EmitProgress sends a type=progress frame.
func (w *Writer) EmitProgress(item Item) error {
	data := map[string]any{
		"type":   "progress",
		"id":     item.ID,
		"layer":  item.Layer,
		"name":   item.Name,
		"status": item.Status,
	}
	if item.Detail != "" {
		data["detail"] = item.Detail
	}
	if item.Hint != "" {
		data["hint"] = item.Hint
	}
	return w.send(data)
}

// EmitMeta sends a type=meta frame with arbitrary key/value fields.
func (w *Writer) EmitMeta(fields map[string]any) error {
	data := map[string]any{"type": "meta"}
	for k, v := range fields {
		data[k] = v
	}
	return w.send(data)
}

// EmitSection sends a type=section frame.
func (w *Writer) EmitSection(title string) error {
	return w.send(map[string]any{
		"type":    "section",
		"message": title,
	})
}

// EmitLog sends a type=log frame. Set verbatim true when the client should print
// message as-is (no CLI prefix); used for server-rendered summary lines.
func (w *Writer) EmitLog(message string, verbatim bool) error {
	data := map[string]any{
		"type":    "log",
		"message": message,
	}
	if verbatim {
		data["verbatim"] = true
	}
	return w.send(data)
}

// EmitDone sends the terminal type=done frame.
func (w *Writer) EmitDone(summary map[string]any) error {
	data := map[string]any{"type": "done"}
	for k, v := range summary {
		data[k] = v
	}
	return w.send(data)
}

// EmitError sends a fatal type=error frame.
func (w *Writer) EmitError(message string) error {
	return w.send(map[string]any{
		"type":    "error",
		"message": message,
	})
}