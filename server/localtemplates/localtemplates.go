// Package localtemplates serves GET /api/local/templates and POST use/add-dir/create
// for the macOS insert picker. Tests inject Store so List never reads $HOME.
package localtemplates

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/xhd2015/dot-pkgs/go-pkgs/fuzzy"
	libtemplates "github.com/xhd2015/my/lib/templates"
)

const (
	// ListPath is GET: ranked templates + missing roots.
	ListPath = "/api/local/templates"
	// UsePath is POST: increment usage for a template .md path.
	UsePath = "/api/local/templates/use"
	// AddDirPath is POST: register a template root directory.
	AddDirPath = "/api/local/templates/add-dir"
	// CreatePath is POST: write a new .md template under a registered root.
	CreatePath = "/api/local/templates/create"
)

// ListResponse is the GET ListPath body.
type ListResponse struct {
	Templates    []TemplateItem `json:"templates"`
	MissingRoots []string       `json:"missing_roots"`
	Roots        []RootItem     `json:"roots"`
}

// RootItem is one registered template root directory.
type RootItem struct {
	Path string `json:"path"`
	Note string `json:"note,omitempty"`
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

// AddDirRequest is the POST AddDirPath body.
type AddDirRequest struct {
	Path string `json:"path"`
	Note string `json:"note,omitempty"`
	// NoteSet is true when the client intends to set/clear note on duplicate.
	NoteSet bool `json:"note_set,omitempty"`
}

// AddDirResponse is the POST AddDirPath body.
type AddDirResponse struct {
	Root      RootItem `json:"root"`
	Duplicate bool     `json:"duplicate"`
}

// CreateRequest is the POST CreatePath body.
type CreateRequest struct {
	Root        string   `json:"root,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Body        string   `json:"body"`
	Filename    string   `json:"filename,omitempty"`
}

// CreateResponse is the POST CreatePath body.
type CreateResponse struct {
	Template libtemplates.Template `json:"template"`
}

// Handler serves local templates endpoints. Nil Store uses DefaultConfigDir.
type Handler struct {
	Store *libtemplates.Store
}

// Register mounts list, use, add-dir, and create on mux.
func Register(mux *http.ServeMux, h *Handler) {
	if h == nil {
		h = &Handler{}
	}
	mux.HandleFunc(ListPath, h.handleList)
	mux.HandleFunc(UsePath, h.handleUse)
	mux.HandleFunc(AddDirPath, h.handleAddDir)
	mux.HandleFunc(CreatePath, h.handleCreate)
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
	roots, err := h.listRoots()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if roots == nil {
		roots = []RootItem{}
	}
	writeJSON(w, http.StatusOK, ListResponse{
		Templates:    filterTemplates(templates, q),
		MissingRoots: missing,
		Roots:        roots,
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

func (h *Handler) handleAddDir(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req AddDirRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	abs, err := normalizeAbsPath(req.Path)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSONError(w, http.StatusBadRequest, "path not found: "+abs)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !info.IsDir() {
		writeJSONError(w, http.StatusBadRequest, "path is not a directory: "+abs)
		return
	}

	configDir, err := h.configDir()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	f, err := libtemplates.LoadFile(configDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	noteSet := req.NoteSet || strings.TrimSpace(req.Note) != ""
	for i := range f.TemplateDirs {
		if f.TemplateDirs[i].Path == abs {
			if noteSet {
				f.TemplateDirs[i].Note = req.Note
				if err := libtemplates.SaveFile(configDir, f); err != nil {
					writeJSONError(w, http.StatusInternalServerError, err.Error())
					return
				}
			}
			writeJSON(w, http.StatusOK, AddDirResponse{
				Root:      RootItem{Path: f.TemplateDirs[i].Path, Note: f.TemplateDirs[i].Note},
				Duplicate: true,
			})
			return
		}
	}
	ent := libtemplates.TemplateDirEntry{
		Path:    abs,
		Note:    req.Note,
		AddedAt: h.now().UTC().Format(time.RFC3339),
	}
	f.TemplateDirs = append(f.TemplateDirs, ent)
	if err := libtemplates.SaveFile(configDir, f); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, AddDirResponse{
		Root:      RootItem{Path: ent.Path, Note: ent.Note},
		Duplicate: false,
	})
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}
	body := req.Body
	if strings.TrimSpace(body) == "" {
		writeJSONError(w, http.StatusBadRequest, "body is required")
		return
	}

	configDir, err := h.configDir()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	f, err := libtemplates.LoadFile(configDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(f.TemplateDirs) == 0 {
		writeJSONError(w, http.StatusBadRequest, "no template roots registered")
		return
	}

	root, err := resolveCreateRoot(f.TemplateDirs, req.Root)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		writeJSONError(w, http.StatusBadRequest, "template root missing or not a directory: "+root)
		return
	}

	filename := strings.TrimSpace(req.Filename)
	if filename == "" {
		filename = slugifyFilename(name)
	}
	filename = filepath.Base(filename)
	if filename == "" || filename == "." || filename == ".." {
		writeJSONError(w, http.StatusBadRequest, "filename is invalid")
		return
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".md") {
		filename += ".md"
	}
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		writeJSONError(w, http.StatusBadRequest, "filename must not contain path separators")
		return
	}

	dest := filepath.Join(root, filename)
	if _, err := os.Stat(dest); err == nil {
		writeJSONError(w, http.StatusConflict, "template already exists: "+dest)
		return
	} else if err != nil && !os.IsNotExist(err) {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	content := renderTemplateMarkdown(name, req.Description, req.Tags, body)
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	templates, _, err := h.store().List()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, t := range templates {
		if t.Path == dest {
			writeJSON(w, http.StatusOK, CreateResponse{Template: t})
			return
		}
	}
	// Fallback if List did not pick it up (should not happen for flat .md).
	writeJSON(w, http.StatusOK, CreateResponse{Template: libtemplates.Template{
		Name:        name,
		FMName:      name,
		Description: strings.TrimSpace(req.Description),
		Tags:        sanitizeTags(req.Tags),
		Path:        dest,
		Body:        strings.TrimRight(body, "\r\n"),
	}})
}

func (h *Handler) listRoots() ([]RootItem, error) {
	configDir, err := h.configDir()
	if err != nil {
		return nil, err
	}
	f, err := libtemplates.LoadFile(configDir)
	if err != nil {
		return nil, err
	}
	out := make([]RootItem, 0, len(f.TemplateDirs))
	for _, d := range f.TemplateDirs {
		out = append(out, RootItem{Path: d.Path, Note: d.Note})
	}
	return out, nil
}

func (h *Handler) configDir() (string, error) {
	st := h.store()
	if st.ConfigDir != "" {
		return st.ConfigDir, nil
	}
	return libtemplates.DefaultConfigDir()
}

func (h *Handler) now() time.Time {
	st := h.store()
	if st.Now != nil {
		return st.Now()
	}
	return time.Now().UTC()
}

func resolveCreateRoot(dirs []libtemplates.TemplateDirEntry, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		if len(dirs) == 1 {
			return dirs[0].Path, nil
		}
		return "", fmt.Errorf("root is required when multiple template roots are registered")
	}
	abs, err := normalizeAbsPath(requested)
	if err != nil {
		return "", err
	}
	for _, d := range dirs {
		if d.Path == abs {
			return abs, nil
		}
	}
	return "", fmt.Errorf("template root not registered: %s", abs)
}

func normalizeAbsPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = home
		} else if strings.HasPrefix(path, "~/") {
			path = filepath.Join(home, path[2:])
		}
	}
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		path = filepath.Join(cwd, path)
	}
	return filepath.Abs(path)
}

var nonSlug = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// slugifyFilename builds a flat .md basename from a display name.
func slugifyFilename(name string) string {
	s := strings.TrimSpace(name)
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return '-'
		}
		return r
	}, s)
	s = nonSlug.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-._")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	if s == "" {
		s = "template"
	}
	if !strings.HasSuffix(strings.ToLower(s), ".md") {
		s += ".md"
	}
	return s
}

func sanitizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func renderTemplateMarkdown(name, description string, tags []string, body string) string {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	tags = sanitizeTags(tags)
	body = strings.TrimRight(body, "\r\n")
	needsFM := name != "" || description != "" || len(tags) > 0
	if !needsFM {
		return body + "\n"
	}
	var b strings.Builder
	b.WriteString("---\n")
	if name != "" {
		b.WriteString("name: ")
		b.WriteString(yamlScalar(name))
		b.WriteByte('\n')
	}
	if description != "" {
		b.WriteString("description: ")
		b.WriteString(yamlScalar(description))
		b.WriteByte('\n')
	}
	if len(tags) > 0 {
		b.WriteString("tags: [")
		for i, t := range tags {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(t)
		}
		b.WriteString("]\n")
	}
	b.WriteString("---\n")
	b.WriteString(body)
	b.WriteByte('\n')
	return b.String()
}

func yamlScalar(v string) string {
	if v == "" {
		return `""`
	}
	needsQuote := strings.ContainsAny(v, ":#{}[],&*!|>'\"%@`") ||
		strings.Contains(v, "\n") ||
		strings.HasPrefix(v, " ") ||
		strings.HasSuffix(v, " ")
	if !needsQuote {
		return v
	}
	escaped := strings.ReplaceAll(v, `"`, `\"`)
	return `"` + escaped + `"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
