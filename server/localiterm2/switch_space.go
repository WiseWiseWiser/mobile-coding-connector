package localiterm2

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type switchSpaceRequest struct {
	SpaceID uint64 `json:"space_id"`
}

func (h *Handler) handleSwitchSpace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req switchSpaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.SwitchSpaceID(req.SpaceID); err != nil {
		if err.Error() == "space_id is required" {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// SwitchSpaceID makes spaceID the current Mission Control Space.
// Must not run inside the macOS app process: SetCurrentSpace is a no-op there.
func (h *Handler) SwitchSpaceID(spaceID uint64) error {
	if spaceID == 0 {
		return fmt.Errorf("space_id is required")
	}
	if h != nil && h.SwitchSpace != nil {
		return h.SwitchSpace(spaceID)
	}
	return liveSwitchSpaceID(spaceID)
}
