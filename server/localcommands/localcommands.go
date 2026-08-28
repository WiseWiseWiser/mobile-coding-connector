// Package localcommands serves GET /api/local/commands and POST use/add endpoints
// for the macOS insert picker. Tests inject Store so List never reads $HOME.
package localcommands

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/xhd2015/dot-pkgs/go-pkgs/fuzzy"
	libcommands "github.com/xhd2015/my/lib/commands"
)

const (
	// ListPath is GET: ranked command bookmarks.
	ListPath = "/api/local/commands"
	// UsePath is POST: increment usage for a registered command.
	UsePath = "/api/local/commands/use"
	// AddPath is POST: register a shell command bookmark.
	AddPath = "/api/local/commands/add"
)

// ListResponse is the GET ListPath body.
type ListResponse struct {
	Commands []CommandItem `json:"commands"`
}

// CommandItem is one picker row, with optional fzf highlight spans when ?q= is set.
type CommandItem struct {
	libcommands.Entry
	Score        int          `json:"score,omitempty"`
	TitleSpans   []fuzzy.Span `json:"title_spans,omitempty"`
	CommandSpans []fuzzy.Span `json:"command_spans,omitempty"`
}

// UseRequest is the POST UsePath body.
type UseRequest struct {
	Command string `json:"command"`
}

// UseResponse is the POST UsePath body.
type UseResponse struct {
	Command libcommands.Entry `json:"command"`
}

// AddRequest is the POST AddPath body.
type AddRequest struct {
	Command string `json:"command"`
	Note    string `json:"note,omitempty"`
	// NoteSet is true when the client intends to set/clear note (including empty).
	// When omitted/false and Note is empty, existing note is left unchanged on duplicate.
	NoteSet bool `json:"note_set,omitempty"`
}

// AddResponse is the POST AddPath body.
type AddResponse struct {
	Command   libcommands.Entry `json:"command"`
	Duplicate bool              `json:"duplicate"`
}

// Handler serves local commands endpoints. Nil Store uses DefaultConfigDir.
type Handler struct {
	Store *libcommands.Store
}

// Register mounts list, use, and add on mux.
func Register(mux *http.ServeMux, h *Handler) {
	if h == nil {
		h = &Handler{}
	}
	mux.HandleFunc(ListPath, h.handleList)
	mux.HandleFunc(UsePath, h.handleUse)
	mux.HandleFunc(AddPath, h.handleAdd)
}

func (h *Handler) store() *libcommands.Store {
	if h != nil && h.Store != nil {
		return h.Store
	}
	return &libcommands.Store{}
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
		entries = []libcommands.Entry{}
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	writeJSON(w, http.StatusOK, ListResponse{
		Commands: filterCommands(entries, q),
	})
}

func filterCommands(entries []libcommands.Entry, q string) []CommandItem {
	tokens := fuzzy.Tokens(q)
	if len(tokens) == 0 {
		ranked := libcommands.Rank(entries)
		out := make([]CommandItem, len(ranked))
		for i, e := range ranked {
			out[i] = CommandItem{Entry: e}
		}
		return out
	}
	out := make([]CommandItem, 0, len(entries))
	for _, e := range entries {
		title := libcommands.Title(e)
		tr := fuzzy.MatchAll(title, tokens)
		cr := fuzzy.MatchAll(e.Command, tokens)
		nr := fuzzy.MatchAll(e.Name, tokens)
		noteR := fuzzy.MatchAll(e.Note, tokens)
		if !tr.OK && !cr.OK && !nr.OK && !noteR.OK {
			continue
		}
		score := 0
		if tr.OK && tr.Score > score {
			score = tr.Score
		}
		if cr.OK && cr.Score > score {
			score = cr.Score
		}
		if nr.OK && nr.Score > score {
			score = nr.Score
		}
		if noteR.OK && noteR.Score > score {
			score = noteR.Score
		}
		item := CommandItem{Entry: e, Score: score}
		if tr.OK {
			item.TitleSpans = tr.Spans
		}
		if cr.OK {
			item.CommandSpans = cr.Spans
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
		return out[i].Command < out[j].Command
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
	command := strings.TrimSpace(req.Command)
	if command == "" {
		writeJSONError(w, http.StatusBadRequest, "command is required")
		return
	}
	ent, err := h.store().RecordUse(command)
	if err != nil {
		if strings.HasPrefix(err.Error(), "command not registered:") || err.Error() == "command is required" {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, UseResponse{Command: *ent})
}

func (h *Handler) handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req AddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	command := strings.TrimSpace(req.Command)
	if command == "" {
		writeJSONError(w, http.StatusBadRequest, "command is required")
		return
	}
	noteSet := req.NoteSet || strings.TrimSpace(req.Note) != ""
	ent, dup, err := h.store().Add(command, req.Note, noteSet)
	if err != nil {
		if err.Error() == "command is required" {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, AddResponse{Command: ent, Duplicate: dup})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
