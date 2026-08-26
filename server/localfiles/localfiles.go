// Package localfiles serves GET /api/local/files and POST /api/local/files/use
// for the macOS insert picker. Tests inject Store so List never reads $HOME.
package localfiles

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/xhd2015/dot-pkgs/go-pkgs/fuzzy"
	"github.com/xhd2015/my/libfiles"
)

const (
	// ListPath is GET: ranked path bookmarks.
	ListPath = "/api/local/files"
	// UsePath is POST: increment usage for a registered path.
	UsePath = "/api/local/files/use"
)

// ListResponse is the GET ListPath body.
type ListResponse struct {
	Files []FileItem `json:"files"`
}

// FileItem is one picker row, with optional fzf highlight spans when ?q= is set.
type FileItem struct {
	libfiles.Entry
	Score      int          `json:"score,omitempty"`
	TitleSpans []fuzzy.Span `json:"title_spans,omitempty"`
	PathSpans  []fuzzy.Span `json:"path_spans,omitempty"`
}

// UseRequest is the POST UsePath body.
type UseRequest struct {
	Path string `json:"path"`
}

// UseResponse is the POST UsePath body.
type UseResponse struct {
	File libfiles.Entry `json:"file"`
}

// Handler serves local files endpoints. Nil Store uses DefaultConfigDir.
type Handler struct {
	Store *libfiles.Store
}

// Register mounts list and use on mux.
func Register(mux *http.ServeMux, h *Handler) {
	if h == nil {
		h = &Handler{}
	}
	mux.HandleFunc(ListPath, h.handleList)
	mux.HandleFunc(UsePath, h.handleUse)
}

func (h *Handler) store() *libfiles.Store {
	if h != nil && h.Store != nil {
		return h.Store
	}
	return &libfiles.Store{}
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	entries, err := h.store().List()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if entries == nil {
		entries = []libfiles.Entry{}
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	writeJSON(w, http.StatusOK, ListResponse{
		Files: filterFiles(entries, q),
	})
}

func filterFiles(entries []libfiles.Entry, q string) []FileItem {
	tokens := fuzzy.Tokens(q)
	if len(tokens) == 0 {
		ranked := libfiles.Rank(entries)
		out := make([]FileItem, len(ranked))
		for i, e := range ranked {
			out[i] = FileItem{Entry: e}
		}
		return out
	}
	out := make([]FileItem, 0, len(entries))
	for _, e := range entries {
		title := libfiles.Title(e)
		tr := fuzzy.MatchAll(title, tokens)
		pr := fuzzy.MatchAll(e.Path, tokens, fuzzy.WithPathScheme())
		nr := fuzzy.MatchAll(e.Name, tokens)
		noteR := fuzzy.MatchAll(e.Note, tokens)
		if !tr.OK && !pr.OK && !nr.OK && !noteR.OK {
			continue
		}
		score := 0
		if tr.OK && tr.Score > score {
			score = tr.Score
		}
		if pr.OK && pr.Score > score {
			score = pr.Score
		}
		if nr.OK && nr.Score > score {
			score = nr.Score
		}
		if noteR.OK && noteR.Score > score {
			score = noteR.Score
		}
		item := FileItem{Entry: e, Score: score}
		if tr.OK {
			item.TitleSpans = tr.Spans
		}
		if pr.OK {
			item.PathSpans = pr.Spans
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].UseCount != out[j].UseCount {
			return out[i].UseCount > out[j].UseCount
		}
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func (h *Handler) handleUse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req UseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		writeJSONError(w, http.StatusBadRequest, "path is required")
		return
	}
	ent, err := h.store().RecordUse(path)
	if err != nil {
		if strings.HasPrefix(err.Error(), "path not registered:") || err.Error() == "path is required" {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, UseResponse{File: *ent})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
