package localiterm2

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type spaceLabelRequest struct {
	SpaceID uint64  `json:"space_id"`
	UUID    string  `json:"uuid"`
	Label   *string `json:"label"`
}

func (h *Handler) handleSpaceLabels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req spaceLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	uuid := strings.TrimSpace(req.UUID)
	if req.SpaceID == 0 && uuid == "" {
		writeJSONError(w, http.StatusBadRequest, "space_id is required")
		return
	}
	label := ""
	if req.Label != nil {
		label = *req.Label
	}

	now := time.Now()
	if h != nil && h.Now != nil {
		now = h.Now()
	}
	if err := h.spaceLabelStore().Set(req.SpaceID, uuid, label, now); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.patchCachedSpaceLabels()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) patchCachedSpaceLabels() {
	if h == nil {
		return
	}
	h.cache.mu.Lock()
	if h.cache.inv == nil {
		h.cache.mu.Unlock()
		return
	}
	next := *h.cache.inv
	h.cache.mu.Unlock()
	next = h.decorateInventory(next)
	h.cache.mu.Lock()
	h.cache.inv = &next
	h.cache.mu.Unlock()
	h.persistInventoryCacheFile(next)
}
