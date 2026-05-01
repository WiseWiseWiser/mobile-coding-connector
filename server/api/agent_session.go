package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/agents"
	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/custom"
	opencode_exposed "github.com/xhd2015/lifelog-private/ai-critic/server/agents/opencode/exposed_opencode"
)

type launchCustomAgentRequest struct {
	ProjectDir string `json:"projectDir"`
	SessionID  string `json:"sessionId,omitempty"`
}

type launchCustomAgentResponse struct {
	SessionID string `json:"sessionId"`
	Port      int    `json:"port"`
	URL       string `json:"url"`
}

func handleLaunchCustomAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req launchCustomAgentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := agents.LaunchCustomAgent(agentID, req.ProjectDir, req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(launchCustomAgentResponse{
		SessionID: result.SessionID,
		Port:      result.Port,
		URL:       result.URL,
	})
}

// handleCustomAgentSessions returns all running custom agent sessions (legacy endpoint).
func handleCustomAgentSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessions := agents.GetCustomAgentSessions()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// handleCustomAgentSessionsByAgent returns all sessions (persisted) for a given agent.
func handleCustomAgentSessionsByAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessions, err := custom.ListSessions(agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	runningIDs := agents.GetRunningCustomSessionIDs()
	for i := range sessions {
		if sessions[i].Status == "running" && !runningIDs[sessions[i].ID] {
			sessions[i].Status = "stopped"
			custom.UpdateSessionStatus(agentID, sessions[i].ID, "stopped", "")
		}
	}

	if sessions == nil {
		sessions = []custom.SessionData{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// handleCustomAgentSessionByID handles operations on a specific session.
func handleCustomAgentSessionByID(w http.ResponseWriter, r *http.Request, sessionID string) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_ = agents.StopCustomAgentSession(sessionID)
	w.WriteHeader(http.StatusNoContent)
}

func handleCustomAgentSessionProxy(w http.ResponseWriter, r *http.Request, sessionID string, rest string) {
	session := agents.GetRunningCustomAgentSession(sessionID)
	if session == nil {
		http.Error(w, "session is not running", http.StatusNotFound)
		return
	}

	restPath := "/"
	rest = strings.TrimPrefix(rest, "/")
	if rest != "" {
		restPath = "/" + rest
	}

	r.URL.Path = restPath
	r.URL.RawPath = ""

	if restPath == "/event" || restPath == "/global/event" {
		opencode_exposed.ProxySSE(w, r, session.Port)
		return
	}
	if strings.Contains(restPath, "/message") && r.Method == http.MethodGet {
		opencode_exposed.ProxyMessages(w, r, session.Port)
		return
	}

	targetURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", session.Port))
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
	}
	proxy.ServeHTTP(w, r)
}
