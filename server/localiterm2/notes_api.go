package localiterm2

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type notesRequest struct {
	SessionID  string  `json:"session_id"`
	Note       *string `json:"note"`
	Bookmarked *bool   `json:"bookmarked"`
}

func (h *Handler) handleNotes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req notesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		writeJSONError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	var lastSeen *NoteLastSeen
	var ident *itemIdentity
	if snap, err := h.captureFn()(); err == nil {
		if sess, ok := FindLiveSession(snap, h.resolveSpaces(snap), sessionID); ok {
			lastSeen = &NoteLastSeen{
				SessionName: sess.SessionName,
				WindowName:  sess.WindowName,
				Cwd:         sess.Cwd,
				SpaceIndex:  sess.SpaceIndex,
			}
			if strings.EqualFold(sess.agentKind, "grok") && sess.agentSessionID != "" {
				ident = &itemIdentity{
					AgentRunner:   sess.agentKind,
					GrokSessionID: sess.agentSessionID,
				}
			}
		}
	}

	now := time.Now()
	if h != nil && h.Now != nil {
		now = h.Now()
	}
	if err := h.noteStore().update(sessionID, req.Note, req.Bookmarked, now, lastSeen, ident); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.patchCachedNotes()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) patchCachedNotes() {
	if h == nil {
		return
	}
	doc := h.noteStore().Document()
	h.cache.mu.Lock()
	if h.cache.inv == nil {
		h.cache.mu.Unlock()
		return
	}
	next := ApplyNotes(*h.cache.inv, doc)
	h.cache.inv = &next
	h.cache.mu.Unlock()
	// Persist notes overlay on complete last-good without recapture.
	h.persistInventoryCacheFile(next)
}
