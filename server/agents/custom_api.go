package agents

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/custom"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

func RegisterCustomAgentsAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/custom-agents", handleCustomAgents)
	mux.HandleFunc("/api/custom-agents/", handleCustomAgent)
}

type CreateCustomAgentRequest struct {
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

type UpdateCustomAgentRequest struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Mode         string            `json:"mode"`
	Model        string            `json:"model,omitempty"`
	Tools        map[string]bool   `json:"tools"`
	Permissions  map[string]string `json:"permissions,omitempty"`
	SystemPrompt string            `json:"systemPrompt,omitempty"`
}

type LaunchCustomAgentRequest struct {
	ProjectDir string `json:"projectDir"`
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

func handleCustomAgent(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if !strings.HasPrefix(path, "/api/custom-agents/") {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	agentID := strings.TrimPrefix(path, "/api/custom-agents/")
	agentID, err := url.PathUnescape(agentID)
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleGetCustomAgent(w, r, agentID)
	case http.MethodPut:
		handleUpdateCustomAgent(w, r, agentID)
	case http.MethodDelete:
		handleDeleteCustomAgent(w, r, agentID)
	case http.MethodPost:
		handleLaunchCustomAgent(w, r, agentID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleListCustomAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := custom.ListAgents()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
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
	type Response struct {
		custom.CustomAgent
		SystemPrompt string `json:"systemPrompt,omitempty"`
	}

	resp := Response{
		CustomAgent:  *agent,
		SystemPrompt: systemPrompt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleCreateCustomAgent(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req CreateCustomAgentRequest
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

	var req UpdateCustomAgentRequest
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

type LaunchCustomAgentResponse struct {
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

	var req LaunchCustomAgentRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	agent, err := custom.LoadAgent(agentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if agent == nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	projectDir := req.ProjectDir
	if info, err := os.Stat(projectDir); err != nil || !info.IsDir() {
		http.Error(w, "Invalid project directory: "+projectDir, http.StatusBadRequest)
		return
	}

	err = custom.GenerateOpencodeConfig(agentID)
	if err != nil {
		http.Error(w, "Failed to generate config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	configDir := custom.GetOpencodeConfigDir(agentID)

	port, err := findFreePort()
	if err != nil {
		http.Error(w, "Failed to find free port", http.StatusInternalServerError)
		return
	}

	cmd, err := tool_exec.New("opencode", []string{"serve", "--port", fmt.Sprintf("%d", port)}, &tool_exec.Options{
		Dir: projectDir,
		Env: map[string]string{
			"OPENCODE_CONFIG_DIR": configDir,
		},
	})
	if err != nil {
		http.Error(w, "Failed to create command: "+err.Error(), http.StatusInternalServerError)
		return
	}

	cmd.Cmd.Env = append(cmd.Cmd.Env, "TERM=xterm-256color")
	cmd.Cmd.Env = tool_resolve.AppendExtraPaths(cmd.Cmd.Env)
	cmd.Cmd.Stdout = os.Stdout
	cmd.Cmd.Stderr = os.Stderr

	if err := cmd.Cmd.Start(); err != nil {
		http.Error(w, "Failed to start agent: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sessionID := fmt.Sprintf("custom-agent-%s-%d", agentID, os.Getpid())

	sessionMgr.mu.Lock()
	sessionMgr.counter++
	sessionMgr.mu.Unlock()

	sessionsMu.Lock()
	if customAgentSessions == nil {
		customAgentSessions = make(map[string]*customAgentSession)
	}
	customAgentSessions[sessionID] = &customAgentSession{
		id:         sessionID,
		agentID:    agentID,
		projectDir: projectDir,
		port:       port,
		cmd:        cmd.Cmd,
	}
	sessionsMu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LaunchCustomAgentResponse{
		SessionID: sessionID,
		Port:      port,
		URL:       fmt.Sprintf("http://127.0.0.1:%d", port),
	})
}

type customAgentSession struct {
	id         string
	agentID    string
	projectDir string
	port       int
	cmd        *exec.Cmd
}

var (
	customAgentSessions map[string]*customAgentSession
	sessionsMu          sync.Mutex
)

func StopCustomAgentSession(sessionID string) error {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	session, ok := customAgentSessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found")
	}

	if session.cmd != nil && session.cmd.Process != nil {
		session.cmd.Process.Kill()
	}

	delete(customAgentSessions, sessionID)
	return nil
}

func GetCustomAgentSessions() []AgentSessionInfo {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	var result []AgentSessionInfo
	for _, s := range customAgentSessions {
		result = append(result, AgentSessionInfo{
			ID:         s.id,
			AgentID:    s.agentID,
			AgentName:  "Custom: " + s.agentID,
			ProjectDir: s.projectDir,
			Port:       s.port,
			CreatedAt:  "",
			Status:     "running",
		})
	}

	slices.SortFunc(result, func(a, b AgentSessionInfo) int {
		if a.AgentID < b.AgentID {
			return -1
		}
		if a.AgentID > b.AgentID {
			return 1
		}
		return 0
	})

	return result
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
