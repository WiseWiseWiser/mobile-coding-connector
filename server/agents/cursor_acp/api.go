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
	mux.HandleFunc("/api/agent/acp/cursor/session/settings", handleSessionSettings)
	mux.HandleFunc("/api/agent/acp/cursor/session/trust-response", handleTrustResponse)
}

func handleSessionSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		sessionID := r.URL.Query().Get("sessionId")
		if sessionID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "sessionId required"})
			return
		}
		agent := getAgent().(*CursorAgent)
		entry := agent.sessionStore.Get(sessionID)
		if entry == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "session not found"})
			return
		}
		resp := struct {
			SessionID      string `json:"sessionId"`
			TrustWorkspace bool   `json:"trustWorkspace"`
			YoloMode       bool   `json:"yoloMode"`
		}{
			SessionID:      sessionID,
			TrustWorkspace: entry.TrustWorkspace,
			YoloMode:       entry.YoloMode,
		}
		json.NewEncoder(w).Encode(resp)

	case http.MethodPost, http.MethodPut:
		var body struct {
			SessionID      string `json:"sessionId"`
			TrustWorkspace *bool  `json:"trustWorkspace,omitempty"`
			YoloMode       *bool  `json:"yoloMode,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}
		if body.SessionID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "sessionId required"})
			return
		}
		agent := getAgent().(*CursorAgent)
		entry := agent.sessionStore.Get(body.SessionID)
		if entry == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "session not found"})
			return
		}
		if body.TrustWorkspace != nil {
			agent.sessionStore.UpdateTrustWorkspace(body.SessionID, *body.TrustWorkspace)
		}
		if body.YoloMode != nil {
			agent.sessionStore.UpdateYoloMode(body.SessionID, *body.YoloMode)
		}
		json.NewEncoder(w).Encode(map[string]string{"message": "Settings updated"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTrustResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		SessionID string `json:"sessionId"`
		Trust     bool   `json:"trust"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	if body.SessionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "sessionId required"})
		return
	}

	agent := getAgent().(*CursorAgent)
	entry := agent.sessionStore.Get(body.SessionID)
	if entry == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "session not found"})
		return
	}

	// Update the trust setting
	agent.sessionStore.UpdateTrustWorkspace(body.SessionID, body.Trust)

	if body.Trust {
		// If trusted, retry the prompt with trust enabled
		go agent.RetryPromptWithTrust(body.SessionID)
		json.NewEncoder(w).Encode(map[string]string{"message": "Trust enabled, retrying prompt"})
	} else {
		json.NewEncoder(w).Encode(map[string]string{"message": "Trust denied"})
	}
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
