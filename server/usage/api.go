package usage

import (
	"encoding/json"
	"net/http"

	"github.com/xhd2015/ai-critic/macosapp/codexusage"
	"github.com/xhd2015/ai-critic/macosapp/debuglog"
	"github.com/xhd2015/ai-critic/macosapp/grokusage"
)

var (
	grokService  = grokusage.NewService()
	codexService = codexusage.NewService()
)

type debugSettingsResponse struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path"`
}

type debugSettingsRequest struct {
	Enabled bool `json:"enabled"`
}

// RegisterAPI registers grok/codex usage and debug log settings on the main server.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/grok/usage", handleGrokUsage)
	mux.HandleFunc("/api/codex/usage", handleCodexUsage)
	mux.HandleFunc("/api/debug/log", handleDebugLog)
}

// Start begins background refresh loops for usage services.
func Start() {
	grokService.Start()
	codexService.Start()
}

// Stop ends background refresh loops for usage services.
func Stop() {
	grokService.Stop()
	codexService.Stop()
}

func handleGrokUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	grokService.EnsureFetch()
	resp := grokService.Get()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleCodexUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	codexService.EnsureFetch()
	resp := codexService.Get()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleDebugLog(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		enabled, path := debuglog.GetSettings()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(debugSettingsResponse{Enabled: enabled, Path: path})
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
				"component": "server",
				"phase":     "settings",
			},
			Fields: map[string]any{"enabled": req.Enabled},
		})
		enabled, path := debuglog.GetSettings()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(debugSettingsResponse{Enabled: enabled, Path: path})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}