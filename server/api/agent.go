package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/custom"
)

func RegisterCustomAgentsAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/custom-agents", handleCustomAgents)
	mux.HandleFunc("/api/custom-agents/", handleCustomAgentRoute)
	mux.HandleFunc("/api/custom-agents/sessions", handleCustomAgentSessions)
}

type createCustomAgentRequest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Mode         string            `json:"mode"`
	Model        string            `json:"model,omitempty"`
	Tools        map[string]bool   `json:"tools"`
	Permissions  map[string]string `json:"permissions,omitempty"`
	Template     string            `json:"template,omitempty"`
	SystemPrompt string            `json:"systemPrompt,omitempty"`
}

type updateCustomAgentRequest struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Mode         string            `json:"mode"`
	Model        string            `json:"model,omitempty"`
	Tools        map[string]bool   `json:"tools"`
	Permissions  map[string]string `json:"permissions,omitempty"`
	SystemPrompt string            `json:"systemPrompt,omitempty"`
}

func handleCustomAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleListCustomAgents(w, r)
	case http.MethodPost:
		handleCreateCustomAgent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleCustomAgentRoute(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if !strings.HasPrefix(path, "/api/custom-agents/") {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	path = strings.TrimPrefix(path, "/api/custom-agents/")
	path, err := url.PathUnescape(path)
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	// Route: /api/custom-agents/sessions/{sessionID}
	// Route: /api/custom-agents/sessions/{sessionID}/proxy/{rest...}
	if strings.HasPrefix(path, "sessions/") {
		sessionPath := strings.TrimPrefix(path, "sessions/")
		if sessionPath != "" {
			parts := strings.SplitN(sessionPath, "/", 2)
			sessionID := parts[0]
			if sessionID == "" {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			if len(parts) == 2 {
				tail := parts[1]
				if tail == "proxy" || strings.HasPrefix(tail, "proxy/") {
					handleCustomAgentSessionProxy(w, r, sessionID, strings.TrimPrefix(tail, "proxy"))
					return
				}
			}
			handleCustomAgentSessionByID(w, r, sessionID)
			return
		}
	}

	// Route: /api/custom-agents/{agentID}/sessions
	if strings.HasSuffix(path, "/sessions") {
		agentID := strings.TrimSuffix(path, "/sessions")
		if agentID != "" {
			handleCustomAgentSessionsByAgent(w, r, agentID)
			return
		}
	}

	var agentID string
	var isLaunch bool
	if strings.HasSuffix(path, "/launch") {
		agentID = strings.TrimSuffix(path, "/launch")
		isLaunch = true
	} else {
		agentID = path
	}

	if agentID == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		if isLaunch {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleGetCustomAgent(w, r, agentID)
	case http.MethodPut:
		if isLaunch {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleUpdateCustomAgent(w, r, agentID)
	case http.MethodDelete:
		if isLaunch {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleDeleteCustomAgent(w, r, agentID)
	case http.MethodPost:
		if !isLaunch {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleLaunchCustomAgent(w, r, agentID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleListCustomAgents(w http.ResponseWriter, r *http.Request) {
	agentList, err := custom.ListAgents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agentList)
}

func handleGetCustomAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	agent, err := custom.LoadAgent(agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if agent == nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	systemPrompt, _ := custom.GetSystemPrompt(agentID)
	type response struct {
		custom.CustomAgent
		SystemPrompt string `json:"systemPrompt,omitempty"`
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response{
		CustomAgent:  *agent,
		SystemPrompt: systemPrompt,
	})
}

func handleCreateCustomAgent(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req createCustomAgentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agentID := req.ID
	if agentID == "" && req.Name != "" {
		agentID = toKebabCase(req.Name)
	}
	if agentID == "" {
		agentID = fmt.Sprintf("agent-%d", os.Getpid())
	}

	existing, _ := custom.LoadAgent(agentID)
	if existing != nil {
		http.Error(w, "Agent already exists", http.StatusConflict)
		return
	}

	cfg := &custom.AgentConfig{
		Name:        req.Name,
		Description: req.Description,
		Mode:        req.Mode,
		Model:       req.Model,
		Tools:       req.Tools,
		Permissions: req.Permissions,
	}
	if cfg.Mode == "" {
		cfg.Mode = "primary"
	}

	if req.Template != "" {
		template := custom.GetTemplate(req.Template)
		if template != nil {
			if cfg.Name == "" {
				cfg.Name = template.Name
			}
			if cfg.Description == "" {
				cfg.Description = template.Description
			}
			if cfg.Mode == "" {
				cfg.Mode = template.Mode
			}
			if len(cfg.Tools) == 0 {
				cfg.Tools = template.Tools
			}
			if len(cfg.Permissions) == 0 {
				cfg.Permissions = template.Permissions
			}
		}
	}

	if err := custom.SaveAgent(agentID, cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.Template != "" {
		template := custom.GetTemplate(req.Template)
		if template != nil && template.SystemPrompt != "" {
			custom.SaveSystemPrompt(agentID, template.SystemPrompt)
		}
	} else if req.SystemPrompt != "" {
		custom.SaveSystemPrompt(agentID, req.SystemPrompt)
	}

	agent, _ := custom.LoadAgent(agentID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

func handleUpdateCustomAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req updateCustomAgentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	existing, err := custom.LoadAgent(agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	cfg := &custom.AgentConfig{
		Name:        req.Name,
		Description: req.Description,
		Mode:        req.Mode,
		Model:       req.Model,
		Tools:       req.Tools,
		Permissions: req.Permissions,
	}
	if cfg.Mode == "" {
		cfg.Mode = "primary"
	}

	if err := custom.SaveAgent(agentID, cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.SystemPrompt != "" {
		custom.SaveSystemPrompt(agentID, req.SystemPrompt)
	}

	agent, _ := custom.LoadAgent(agentID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}

func handleDeleteCustomAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	existing, err := custom.LoadAgent(agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if existing == nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	if err := custom.DeleteAgent(agentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toKebabCase(s string) string {
	var result []rune
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result = append(result, '-')
			}
			result = append(result, r+32)
		} else if r == ' ' || r == '_' {
			result = append(result, '-')
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
