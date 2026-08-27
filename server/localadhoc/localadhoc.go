// Package localadhoc serves the insert-picker adhoc text pad (persisted file).
package localadhoc

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Path is GET (read) / PUT (write) for adhoc text.
	Path = "/api/local/adhoc"
	fileName = "insert-adhoc.txt"
)

// Response is the GET/PUT body.
type Response struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

// PutRequest is the PUT body.
type PutRequest struct {
	Content string `json:"content"`
}

// Handler serves adhoc endpoints. Nil DataDir uses ~/.ai-critic.
type Handler struct {
	DataDir string
}

// Register mounts the adhoc path on mux.
func Register(mux *http.ServeMux, h *Handler) {
	if h == nil {
		h = &Handler{}
	}
	mux.HandleFunc(Path, h.handle)
}

func (h *Handler) filePath() (string, error) {
	dir := ""
	if h != nil {
		dir = strings.TrimSpace(h.DataDir)
	}
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".ai-critic")
	}
	return filepath.Join(dir, fileName), nil
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPut:
		h.handlePut(w, r)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) handleGet(w http.ResponseWriter, r *http.Request) {
	path, err := h.filePath()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusOK, Response{Content: "", Path: path})
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, Response{Content: string(data), Path: path})
}

func (h *Handler) handlePut(w http.ResponseWriter, r *http.Request) {
	var req PutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	path, err := h.filePath()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := os.WriteFile(path, []byte(req.Content), 0o644); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, Response{Content: req.Content, Path: path})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
