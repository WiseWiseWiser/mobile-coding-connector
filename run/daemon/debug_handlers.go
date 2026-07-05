package daemon

import (
	"encoding/json"
	"net/http"

	"github.com/xhd2015/ai-critic/macosapp/debuglog"
)

type debugSettingsResponse struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
}

type debugSettingsRequest struct {
	Enabled bool `json:"enabled"`
}

func (s *HTTPServer) handleDebugSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		enabled, path := debuglog.GetSettings()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(debugSettingsResponse{Enabled: enabled, Path: path})
	case http.MethodPut:
		var req debugSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := debuglog.SetEnabled(req.Enabled); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		debuglog.Write(debuglog.Entry{
			Event: "debug_toggled",
			Labels: map[string]string{
				"component": "keepalive",
				"phase":     "settings",
			},
			Fields: map[string]any{"enabled": req.Enabled},
		})
		enabled, path := debuglog.GetSettings()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(debugSettingsResponse{Enabled: enabled, Path: path})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}