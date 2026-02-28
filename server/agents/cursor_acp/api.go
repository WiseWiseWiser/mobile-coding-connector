package cursor_acp

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/acp"
)

var (
	globalAgent     *CursorAgent
	globalAgentLock sync.Mutex
)

func getAgent() acp.Agent {
	globalAgentLock.Lock()
	defer globalAgentLock.Unlock()
	if globalAgent == nil {
		globalAgent = NewCursorAgent()
	}
	return globalAgent
}

func RegisterAPI(mux *http.ServeMux) {
	acp.RegisterAgentAPI(mux, "/api/agent/acp/cursor", getAgent())

	mux.HandleFunc("/api/agent/acp/cursor/settings", handleSettings)
}

func handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		settings, err := LoadSettings()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		resp := struct {
			*CursorAgentSettings
			EffectivePath EffectivePathInfo `json:"effective_path"`
		}{
			CursorAgentSettings: settings,
			EffectivePath:       ResolveEffectivePath(),
		}
		json.NewEncoder(w).Encode(resp)

	case http.MethodPost, http.MethodPut:
		var settings CursorAgentSettings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}
		if err := SaveSettings(&settings); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "Settings saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
