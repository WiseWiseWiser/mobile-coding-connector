// Package localskills serves GET /api/local/skills and POST /api/local/skills/use
// for the macOS skills picker. Tests inject Store so List never reads $HOME.
package localskills

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/xhd2015/dot-pkgs/go-pkgs/fuzzy"
	"github.com/xhd2015/my/libskills"
)

const (
	// ListPath is GET: ranked skills + missing roots.
	ListPath = "/api/local/skills"
	// UsePath is POST: increment usage for a SKILL.md path.
	UsePath = "/api/local/skills/use"
)

// ListResponse is the GET ListPath body.
type ListResponse struct {
	Skills       []SkillItem `json:"skills"`
	MissingRoots []string    `json:"missing_roots"`
}

// SkillItem is one picker row, with optional fzf highlight spans when ?q= is set.
type SkillItem struct {
	libskills.Skill
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
	Skill libskills.Skill `json:"skill"`
}

// Handler serves local skills endpoints. Nil Store uses DefaultConfigDir.
type Handler struct {
	Store *libskills.Store
}

// Register mounts list and use on mux.
func Register(mux *http.ServeMux, h *Handler) {
	if h == nil {
		h = &Handler{}
	}
	mux.HandleFunc(ListPath, h.handleList)
	mux.HandleFunc(UsePath, h.handleUse)
}

func (h *Handler) store() *libskills.Store {
	if h != nil && h.Store != nil {
		return h.Store
	}
	return &libskills.Store{}
}

func (h *Handler) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	skills, missing, err := h.store().List()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if skills == nil {
		skills = []libskills.Skill{}
	}
	if missing == nil {
		missing = []string{}
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	writeJSON(w, http.StatusOK, ListResponse{
		Skills:       filterSkills(skills, q),
		MissingRoots: missing,
	})
}

func skillTitle(s libskills.Skill) string {
	if strings.TrimSpace(s.FMName) != "" {
		return s.FMName
	}
	return s.Name
}

func filterSkills(skills []libskills.Skill, q string) []SkillItem {
	tokens := fuzzy.Tokens(q)
	if len(tokens) == 0 {
		ranked := libskills.Rank(skills)
		out := make([]SkillItem, len(ranked))
		for i, s := range ranked {
			out[i] = SkillItem{Skill: s}
		}
		return out
	}
	out := make([]SkillItem, 0, len(skills))
	for _, s := range skills {
		title := skillTitle(s)
		tr := fuzzy.MatchAll(title, tokens)
		pr := fuzzy.MatchAll(s.Path, tokens, fuzzy.WithPathScheme())
		nr := fuzzy.MatchAll(s.Name, tokens)
		if !tr.OK && !pr.OK && !nr.OK {
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
		item := SkillItem{Skill: s, Score: score}
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
	sk, err := h.store().RecordUse(path)
	if err != nil {
		if strings.HasPrefix(err.Error(), "skill not found:") || err.Error() == "path is required" {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, UseResponse{Skill: *sk})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
