// Package screen streams this machine's desktop to a browser and injects the
// viewer's mouse and keyboard input back into the host. It is the transport
// behind the /screen page: ffmpeg captures MJPEG, changed 128x128 tiles are
// re-encoded as JPEG, and input travels over the same WebSocket.
package screen

import (
	"encoding/json"
	"net/http"
)

// RegisterAPI wires the screen endpoints into mux.
func RegisterAPI(mux *http.ServeMux) {
	m := Default()

	mux.HandleFunc("GET /api/screen/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, m.Snapshot())
	})
	mux.HandleFunc("GET /api/screen/config", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, m.Config())
	})
	mux.HandleFunc("POST /api/screen/config", func(w http.ResponseWriter, r *http.Request) {
		// Decode over the live config so a partial body (for example just
		// {"fps": 20}) cannot silently switch capture or control off.
		cfg := m.Config()
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid config: "+err.Error())
			return
		}
		if err := m.UpdateConfig(cfg); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, m.Config())
	})
	// Admin kill switch: drop whoever currently controls the desktop.
	mux.HandleFunc("POST /api/screen/control/release", func(w http.ResponseWriter, r *http.Request) {
		m.ReleaseControl("released by admin")
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	// macOS shows no prompt we can answer for the user: open the exact settings
	// pane and ask the system to register the app in the list.
	mux.HandleFunc("POST /api/screen/permission/prompt", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Which string `json:"which"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if err := promptPermission(req.Which); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "input": m.inputInfo()})
	})
	mux.HandleFunc("/api/screen/ws", m.handleWS)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}
