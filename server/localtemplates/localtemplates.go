// Package localtemplates serves GET /api/local/templates and POST /api/local/templates/use
// for the macOS insert picker. Tests inject Store so List never reads $HOME.
package localtemplates

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/xhd2015/dot-pkgs/go-pkgs/fuzzy"
	"github.com/xhd2015/my/libtemplates"
)

const (
	// ListPath is GET: ranked templates + missing roots.
	ListPath = "/api/local/templates"
	// UsePath is POST: increment usage for a template .md path.
	UsePath = "/api/local/templates/use"
)

// ListResponse is the GET ListPath body.
type ListResponse struct {
	Templates    []TemplateItem `json:"templates"`
	MissingRoots []string       `json:"missing_roots"`
}

// TemplateItem is one picker row, with optional fzf highlight spans when ?q= is set.
type TemplateItem struct {
	libtemplates.Template
	Score      int          `json:"score,omitempty"`
	TitleSpans []fuzzy.Span `json:"title_spans,omitempty"`
	PathSpans  []fuzzy.Span `json:"path_spans,omitempty"`
	BodySpans  []fuzzy.Span `json:"body_spans,omitempty"`
}

// UseRequest is the POST UsePath body.
type UseRequest struct {
	Path string `json:"path"`
}

// UseResponse is the POST UsePath body.
type UseResponse struct {
	Template libtemplates.Template `json:"template"`
}

// Handler serves local templates endpoints. Nil Store uses DefaultConfigDir.
type Handler struct {
	Store *libtemplates.Store
}

// Register mounts list and use on mux.
func Register(mux *http.ServeMux, h *Handler) {
	if h == nil {
		h = &Handler{}
	}
	mux.HandleFunc(ListPath, h.handleList)
	mux.HandleFunc(UsePath, h.handleUse)
}

func (h *Handler) store() *libtemplates.Store {
	if h != nil && h.Store != nil {
		return h.Store
	}
	return &libtemplates.Store{}
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	templates, missing, err := h.store().List()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if templates == nil {
		templates = []libtemplates.Template{}
	}
	if missing == nil {
		missing = []string{}
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	writeJSON(w, http.StatusOK, ListResponse{
		Templates:    filterTemplates(templates, q),
		MissingRoots: missing,
	})
}

func filterTemplates(templates []libtemplates.Template, q string) []TemplateItem {
	tokens := fuzzy.Tokens(q)
	if len(tokens) == 0 {
		ranked := libtemplates.Rank(templates)
		out := make([]TemplateItem, len(ranked))
		for i, t := range ranked {
			out[i] = TemplateItem{Template: t}
		}
		return out
	}
	out := make([]TemplateItem, 0, len(templates))
	for _, t := range templates {
		title := libtemplates.Title(t)
		tr := fuzzy.MatchAll(title, tokens)
		pr := fuzzy.MatchAll(t.Path, tokens, fuzzy.WithPathScheme())
		nr := fuzzy.MatchAll(t.Name, tokens)
		dr := fuzzy.MatchAll(t.Description, tokens)
		br := fuzzy.MatchAll(t.Body, tokens)
		tagOK := false
		for _, tag := range t.Tags {
			if fuzzy.MatchAll(tag, tokens).OK {
				tagOK = true
				break
			}
		}
		if !tr.OK && !pr.OK && !nr.OK && !dr.OK && !br.OK && !tagOK {
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
		if dr.OK && dr.Score > score {
			score = dr.Score
		}
		if br.OK && br.Score > score {
			score = br.Score
		}
		item := TemplateItem{Template: t, Score: score}
		if tr.OK {
			item.TitleSpans = tr.Spans
		}
		if pr.OK {
			item.PathSpans = pr.Spans
		}
		if br.OK {
			item.BodySpans = br.Spans
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
	tmpl, err := h.store().RecordUse(path)
	if err != nil {
		if strings.HasPrefix(err.Error(), "template not found:") || err.Error() == "path is required" {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, UseResponse{Template: *tmpl})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
