package bookmarks

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
)

var (
	defaultManager     *Manager
	defaultManagerOnce sync.Once
	defaultManagerErr  error
)

// RegisterAPI registers bookmarks endpoints on the production default store path.
func RegisterAPI(mux *http.ServeMux) {
	defaultManagerOnce.Do(func() {
		path, err := DefaultPath()
		if err != nil {
			defaultManagerErr = err
			return
		}
		m := NewManagerAt(path)
		if _, err := m.Load(); err != nil {
			defaultManagerErr = err
			return
		}
		defaultManager = m
	})
	RegisterAPIWithManager(mux, defaultManager)
}

// RegisterAPIWithManager mounts bookmarks HTTP handlers using the given manager.
func RegisterAPIWithManager(mux *http.ServeMux, m *Manager) {
	registerAPI(mux, m, nil)
}

// RegisterAPIWithChrome mounts the app-local API plus the read-only Chrome
// source. A nil reader resolves Chrome's default profile per request.
func RegisterAPIWithChrome(mux *http.ServeMux, m *Manager, chrome *ChromeReader) {
	registerAPI(mux, m, chrome)
}

func registerAPI(mux *http.ServeMux, m *Manager, chrome *ChromeReader) {
	mux.HandleFunc("/api/bookmarks", func(w http.ResponseWriter, r *http.Request) {
		source := requestedBookmarkSource(r)
		if source == "chrome" {
			handleChromeBookmarks(w, r, chrome)
			return
		}
		if source != "" {
			writeJSONError(w, http.StatusBadRequest, "source must be empty or chrome")
			return
		}
		if m == nil {
			if defaultManagerErr != nil {
				writeJSONError(w, http.StatusInternalServerError, defaultManagerErr.Error())
				return
			}
			writeJSONError(w, http.StatusInternalServerError, "bookmarks manager not available")
			return
		}
		handleBookmarks(w, r, m)
	})
	mux.HandleFunc("/api/bookmarks/move", func(w http.ResponseWriter, r *http.Request) {
		source := requestedBookmarkSource(r)
		if source == "chrome" {
			writeJSONError(w, http.StatusBadRequest, "Chrome bookmarks are read-only")
			return
		}
		if source != "" {
			writeJSONError(w, http.StatusBadRequest, "source must be empty or chrome")
			return
		}
		if m == nil {
			writeJSONError(w, http.StatusInternalServerError, "bookmarks manager not available")
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleMove(w, r, m)
	})
}

func requestedBookmarkSource(r *http.Request) string {
	return strings.ToLower(strings.TrimSpace(r.URL.Query().Get("source")))
}

func handleChromeBookmarks(w http.ResponseWriter, r *http.Request, chrome *ChromeReader) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusBadRequest, "Chrome bookmarks are read-only")
		return
	}
	reader := chrome
	if reader == nil {
		var err error
		reader, err = DefaultChromeReader()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Chrome profile is not available")
			return
		}
	}
	doc, err := reader.Document()
	if err != nil {
		switch {
		case errors.Is(err, ErrChromeInvalid):
			writeJSONError(w, http.StatusInternalServerError, "Chrome bookmarks could not be parsed")
		case errors.Is(err, ErrChromeUnreadable):
			writeJSONError(w, http.StatusInternalServerError, "Chrome bookmarks are not readable")
		default:
			writeJSONError(w, http.StatusInternalServerError, "Chrome bookmarks are not available")
		}
		return
	}
	writeJSON(w, doc)
}

type addRequest struct {
	ParentID string  `json:"parent_id"`
	Type     string  `json:"type"`
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	URL      string  `json:"url"`
	Browser  *string `json:"browser"`
	Index    *int    `json:"index"`
}

type patchRequest struct {
	Name    *string `json:"name"`
	URL     *string `json:"url"`
	Browser *string `json:"browser"`
}

type moveRequest struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	Index    *int   `json:"index"`
}

func handleBookmarks(w http.ResponseWriter, r *http.Request, m *Manager) {
	switch r.Method {
	case http.MethodGet:
		doc := m.Document()
		if doc == nil {
			if _, err := m.Load(); err != nil {
				writeJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}
			doc = m.Document()
		}
		writeJSON(w, doc)
	case http.MethodPost:
		var req addRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		n := &Node{
			Type:    req.Type,
			ID:      req.ID,
			Name:    req.Name,
			URL:     req.URL,
			Browser: req.Browser,
		}
		added, err := m.Add(req.ParentID, n, req.Index)
		if err != nil {
			writeManagerError(w, err)
			return
		}
		writeJSON(w, added)
	case http.MethodPatch:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}
		var req patchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		opts := UpdateOpts{
			Name: req.Name,
			URL:  req.URL,
		}
		if req.Browser != nil {
			if strings.TrimSpace(*req.Browser) == "" {
				opts.ClearBrowser = true
			} else {
				opts.Browser = req.Browser
			}
		}
		updated, err := m.Update(id, opts)
		if err != nil {
			writeManagerError(w, err)
			return
		}
		writeJSON(w, updated)
	case http.MethodDelete:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "id is required")
			return
		}
		if err := m.Delete(id); err != nil {
			writeManagerError(w, err)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleMove(w http.ResponseWriter, r *http.Request, m *Manager) {
	var req moveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		writeJSONError(w, http.StatusBadRequest, "id is required")
		return
	}
	if err := m.Move(req.ID, req.ParentID, req.Index); err != nil {
		writeManagerError(w, err)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func writeManagerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeJSONError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrBadParent):
		writeJSONError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidName),
		errors.Is(err, ErrInvalidURL),
		errors.Is(err, ErrInvalidType),
		errors.Is(err, ErrInvalidBrowser),
		errors.Is(err, ErrParentNotFolder),
		errors.Is(err, ErrCannotDeleteRoot):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	default:
		msg := err.Error()
		if strings.Contains(msg, "already exists") ||
			strings.Contains(msg, "cannot move") ||
			strings.Contains(msg, "required") {
			writeJSONError(w, http.StatusBadRequest, msg)
			return
		}
		writeJSONError(w, http.StatusInternalServerError, msg)
	}
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
