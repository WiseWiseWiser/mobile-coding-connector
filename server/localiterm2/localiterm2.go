// Package localiterm2 serves POST /api/local/iterm2/open for local menu-bar
// clients that open directories in iTerm2 via shell/iterm2.OpenConfig.
package localiterm2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2"
)

// Path is the fixed product endpoint for local iTerm2 open.
const Path = "/api/local/iterm2/open"

// openRequest is the JSON body for Path.
type openRequest struct {
	Dir  string   `json:"dir"`
	Mode string   `json:"mode"`
	Send []string `json:"send"`
}

// Handler serves Path. Open is injectible for tests; when nil, defaults to
// iterm2.OpenConfig.
type Handler struct {
	Open func(dir string, cfg *iterm2.Config) error
}

// ParseOpenMode maps JSON mode strings to iterm2.OpenMode.
// Empty / "reuse" → ModeReuseCurrent; "new" → ModeForceNew; "smart" → ModeSmart.
// Unknown values return an error (do not fall through to ModeSmart zero-value).
func ParseOpenMode(s string) (iterm2.OpenMode, error) {
	switch strings.TrimSpace(s) {
	case "", "reuse":
		return iterm2.ModeReuseCurrent, nil
	case "new":
		return iterm2.ModeForceNew, nil
	case "smart":
		return iterm2.ModeSmart, nil
	default:
		return 0, fmt.Errorf("invalid open mode %q (want reuse, new, or smart)", s)
	}
}

// Register mounts h on mux at Path (all methods; ServeHTTP enforces POST).
func Register(mux *http.ServeMux, h *Handler) {
	if h == nil {
		h = &Handler{}
	}
	mux.Handle(Path, h)
}

func (h *Handler) openFunc() func(dir string, cfg *iterm2.Config) error {
	if h != nil && h.Open != nil {
		return h.Open
	}
	return iterm2.OpenConfig
}

// ServeHTTP handles POST /api/local/iterm2/open.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req openRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dir := strings.TrimSpace(req.Dir)
	if dir == "" {
		writeJSONError(w, http.StatusBadRequest, "dir is required")
		return
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("dir does not exist: %s", dir))
			return
		}
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("dir: %v", err))
		return
	}
	if !info.IsDir() {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("not a directory: %s", dir))
		return
	}

	mode, err := ParseOpenMode(req.Mode)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	cfg := &iterm2.Config{
		Mode:             mode,
		FollowUpCommands: req.Send,
	}
	if err := h.openFunc()(dir, cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
