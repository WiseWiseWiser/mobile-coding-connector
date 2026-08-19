package localiterm2

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	shelliterm "github.com/xhd2015/dot-pkgs/go-pkgs/shell/iterm2"
	kooliterm "github.com/xhd2015/kool/tools/iterm2"
)

type focusRequest struct {
	SessionID string `json:"session_id"`
}

func (h *Handler) handleFocus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req focusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		writeJSONError(w, http.StatusBadRequest, "session_id is required")
		return
	}

	if err := h.FocusSession(sessionID); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "required") {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// FocusSession focuses a live iTerm pane by session id (library path, no HTTP).
// Unknown ids return an error matching (?i)not found|session.
func (h *Handler) FocusSession(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	sess, ok := h.liveFromCache(sessionID)
	if !ok {
		snap, err := h.captureNoEnrich()
		if err != nil {
			if isITermNotRunning(err) {
				return fmt.Errorf("session not found")
			}
			return err
		}
		sess, ok = FindLiveSession(snap, h.resolveSpaces(snap), sessionID)
		if !ok {
			return fmt.Errorf("session not found")
		}
	}

	ref := shelliterm.SessionRef{
		WindowID:  sess.WindowID,
		TabIndex:  sess.TabIndex,
		SessionID: sess.SessionID,
	}
	if h != nil && h.Focus != nil {
		return h.focusFn()(ref)
	}
	return focusSessionByID(sess.SessionID)
}

func focusSessionByID(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	escaped := strings.ReplaceAll(sessionID, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	script := `tell application "iTerm2"
  activate
  set theID to "` + escaped + `"
  set winCount to count of windows
  repeat with w from 1 to winCount
    try
      set tabCount to count of tabs of window w
      repeat with t from 1 to tabCount
        try
          set sessCount to count of sessions of tab t of window w
          repeat with s from 1 to sessCount
            try
              if (id of session s of tab t of window w as string) is theID then
                select window w
                select tab t of window w
                select session s of tab t of window w
              end if
            end try
          end repeat
        end try
      end repeat
    end try
  end repeat
end tell`
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return fmt.Errorf("osascript: %s", msg)
		}
		return err
	}
	return nil
}

func (h *Handler) liveFromCache(sessionID string) (LiveSession, bool) {
	inv, ok := h.cachedInventory()
	if !ok {
		return LiveSession{}, false
	}
	for _, g := range inv.Desktops {
		for _, sess := range g.Sessions {
			if sess.SessionID == sessionID {
				return sess, true
			}
		}
	}
	return LiveSession{}, false
}

func (h *Handler) captureNoEnrich() (*kooliterm.Snapshot, error) {
	if h != nil && h.Capture != nil {
		return h.Capture()
	}
	snap, _, err := kooliterm.CaptureSnapshotWith(kooliterm.CaptureOpts{NoEnrich: true})
	return snap, err
}
