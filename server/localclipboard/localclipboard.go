// Package localclipboard serves clipboard peek/dump for the macOS insert picker.
package localclipboard

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/xhd2015/dot-pkgs/go-pkgs/getclipboard"
)

const (
	// PeekPath is GET: non-destructive clipboard summary.
	PeekPath = "/api/local/clipboard/peek"
	// DumpPath is POST: write clipboard to a temp file and return the path.
	DumpPath = "/api/local/clipboard/dump"
)

// PeekResponse is the GET PeekPath body.
type PeekResponse struct {
	Kind      string   `json:"kind"`
	Ext       string   `json:"ext,omitempty"`
	Bytes     int      `json:"bytes"`
	Preview   string   `json:"preview,omitempty"`
	MIME      string   `json:"mime,omitempty"`
	Available []string `json:"available,omitempty"`
}

// DumpRequest is the optional POST DumpPath body.
type DumpRequest struct {
	Name   string `json:"name,omitempty"`
	Output string `json:"output,omitempty"`
}

// DumpResponse is the POST DumpPath body.
type DumpResponse struct {
	Path  string `json:"path"`
	Kind  string `json:"kind"`
	Ext   string `json:"ext"`
	Bytes int    `json:"bytes"`
}

// Handler serves clipboard endpoints. Nil Source uses the system clipboard.
type Handler struct {
	Source getclipboard.Source
	// DumpDir overrides /tmp for dump output (tests).
	DumpDir string
}

// Register mounts peek and dump on mux.
func Register(mux *http.ServeMux, h *Handler) {
	if h == nil {
		h = &Handler{}
	}
	mux.HandleFunc(PeekPath, h.handlePeek)
	mux.HandleFunc(DumpPath, h.handleDump)
}

func (h *Handler) source() getclipboard.Source {
	if h != nil && h.Source != nil {
		return h.Source
	}
	return getclipboard.System()
}

func (h *Handler) handlePeek(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	peek, err := getclipboard.Peek(h.source(), getclipboard.DefaultPreviewMax)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, PeekResponse{
		Kind:      string(peek.Kind),
		Ext:       peek.Ext,
		Bytes:     peek.Bytes,
		Preview:   peek.Preview,
		MIME:      peek.MIME,
		Available: peek.Available,
	})
}

func (h *Handler) handleDump(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req DumpRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	dir := "/tmp"
	if h != nil && strings.TrimSpace(h.DumpDir) != "" {
		dir = h.DumpDir
	}
	res, err := getclipboard.Dump(h.source(), getclipboard.DumpOptions{
		Output: strings.TrimSpace(req.Output),
		Name:   strings.TrimSpace(req.Name),
		Dir:    dir,
	})
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "empty") || strings.Contains(msg, "unsupported") {
			writeJSONError(w, http.StatusConflict, msg)
			return
		}
		if strings.Contains(msg, "mutually exclusive") || strings.Contains(msg, "--name must") {
			writeJSONError(w, http.StatusBadRequest, msg)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, msg)
		return
	}
	writeJSON(w, http.StatusOK, DumpResponse{
		Path:  res.Path,
		Kind:  string(res.Kind),
		Ext:   res.Ext,
		Bytes: res.Bytes,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
