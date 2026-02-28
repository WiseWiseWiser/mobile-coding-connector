package agents

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/cursor"
	opencode_exposed "github.com/xhd2015/lifelog-private/ai-critic/server/agents/opencode/exposed_opencode"
	opencode_internal "github.com/xhd2015/lifelog-private/ai-critic/server/agents/opencode/internal_opencode"
	"github.com/xhd2015/lifelog-private/ai-critic/server/settings"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

// AgentDef defines a supported coding agent
type AgentID string

const (
	AgentIDOpenCode    AgentID = "opencode"
	AgentIDClaudeCode  AgentID = "claude-code"
	AgentIDCodex       AgentID = "codex"
	AgentIDCursorAgent AgentID = "cursor-agent"
)

type AgentDef struct {
	ID          AgentID `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Command     string  `json:"command"`
	// Installed is set dynamically by checking if the command is available
	Installed bool `json:"installed"`
	// Headless indicates this agent supports headless server mode
	Headless bool `json:"headless"`
}

// All supported agents - add new agents here
var agentDefs = []AgentDef{
	{
		ID:          AgentIDOpenCode,
		Name:        "OpenCode",
		Description: "AI-powered coding assistant with headless server mode",
		Command:     "opencode",
		Headless:    true,
	},
	{
		ID:          AgentIDClaudeCode,
		Name:        "Claude Code",
		Description: "Anthropic's Claude coding agent (CLI)",
		Command:     "claude",
	},
	{
		ID:          AgentIDCodex,
		Name:        "Codex",
		Description: "OpenAI Codex CLI agent",
		Command:     "codex",
	},
	{
		ID:          AgentIDCursorAgent,
		Name:        "Cursor Agent",
		Description: "Cursor's AI coding agent (chat mode via stream-json adapter)",
		Command:     "cursor-agent",
		Headless:    true,
	},
}

// AgentSessionInfo is returned to the frontend
type AgentSessionInfo struct {
	ID         string `json:"id"`
	AgentID    string `json:"agent_id"`
	AgentName  string `json:"agent_name"`
	ProjectDir string `json:"project_dir"`
	Port       int    `json:"port"`
	CreatedAt  string `json:"created_at"`
	Status     string `json:"status"` // "starting", "running", "stopped", "error"
	Error      string `json:"error,omitempty"`
}

// AgentSessionsResponse holds paginated agent sessions response
type AgentSessionsResponse struct {
	Sessions   []AgentSessionInfo `json:"sessions"`
	Page       int                `json:"page"`
	PageSize   int                `json:"page_size"`
	Total      int                `json:"total"`
	TotalPages int                `json:"total_pages"`
}

// agentSession holds state for a running headless agent process
type agentSession struct {
	id         string
	agentID    string
	agentName  string
	projectDir string
	port       int
	createdAt  time.Time
	cmd        *exec.Cmd
	proxy      *httputil.ReverseProxy

	// For cursor-agent adapter mode (no external HTTP server, handled in-process)
	cursorAdapter *cursor.Adapter

	mu     sync.Mutex
	status string // "starting", "running", "stopped", "error"
	err    string
	done   chan struct{}
}

type agentSessionManager struct {
	mu            sync.Mutex
	sessions      map[string]*agentSession
	counter       int
	settingsStore *settings.Store
}

var sessionMgr = newSessionManager()

func newSessionManager() *agentSessionManager {
	store, _ := settings.NewStore(".settings")
	return &agentSessionManager{
		sessions:      make(map[string]*agentSession),
		settingsStore: store,
	}
}

// RegisterAPI registers agent-related API endpoints
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/agents", handleListAgents)
	mux.HandleFunc("/api/agents/config", handleAgentConfig)
	mux.HandleFunc("/api/agents/effective-path", handleAgentEffectivePath)
	mux.HandleFunc("/api/agents/opencode/auth", handleOpencodeAuth)
	mux.HandleFunc("/api/agents/opencode/settings", handleOpencodeSettings)
	mux.HandleFunc("/api/agents/opencode/web-status", handleOpencodeWebStatus)
	mux.HandleFunc("/api/agents/opencode/server", handleOpencodeServer)
	mux.HandleFunc("/api/agents/opencode/exposed-server/start", handleOpencodeWebServerStart)
	mux.HandleFunc("/api/agents/opencode/exposed-server/start/stream", handleOpencodeWebServerStartStreaming)
	mux.HandleFunc("/api/agents/opencode/exposed-server/stop", handleOpencodeWebServerStop)
	mux.HandleFunc("/api/agents/opencode/exposed-server/stop/stream", handleOpencodeWebServerStopStreaming)
	mux.HandleFunc("/api/agents/opencode/web-server/domain-map", handleOpencodeWebServerDomainMap)
	mux.HandleFunc("/api/agents/opencode/web-server/domain-map/stream", handleOpencodeWebServerDomainMapStreaming)
	mux.HandleFunc("/api/agents/sessions", handleAgentSessions)
	// Proxy: /api/agents/sessions/{sessionID}/proxy/... -> opencode server
	mux.HandleFunc("/api/agents/sessions/", handleAgentSessionProxy)
	// External opencode sessions (from CLI/web)
	mux.HandleFunc("/api/agents/external-sessions", handleExternalSessions)

	// Custom agents API
	RegisterCustomAgentsAPI(mux)
}

// Shutdown stops the agents module (stops health checks, but leaves opencode running)
func Shutdown() {
	fmt.Println("Stopping opencode health check...")
	opencode_exposed.StopHealthCheck()
}

// ------ Agent Session Manager ------

func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port, nil
}

func (m *agentSessionManager) launch(agentID, projectDir, apiKey string) (*agentSession, error) {
	aid := AgentID(agentID)
	// Find the agent def
	var agentDef *AgentDef
	for i := range agentDefs {
		if agentDefs[i].ID == aid {
			agentDef = &agentDefs[i]
			break
		}
	}
	if agentDef == nil {
		return nil, fmt.Errorf("unknown agent: %s", agentID)
	}
	if !agentDef.Headless {
		return nil, fmt.Errorf("agent %s does not support headless mode", agentID)
	}

	// Validate project dir
	if info, err := os.Stat(projectDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("invalid project directory: %s", projectDir)
	}

	m.mu.Lock()
	m.counter++
	id := fmt.Sprintf("agent-session-%d", m.counter)
	m.mu.Unlock()

	// For cursor-agent, use the in-process adapter instead of an external HTTP server
	if agentDef.ID == AgentIDCursorAgent {
		return m.launchCursorAdapter(id, agentDef, projectDir, apiKey)
	}

	// Check command is installed and get full path (considering custom binary path)
	cmdPath, err := getAgentBinaryPath(agentDef.ID, agentDef.Command)
	if err != nil {
		return nil, fmt.Errorf("agent %s is not installed (%s not found)", agentDef.Name, agentDef.Command)
	}

	// Find a free port
	port, err := findFreePort()
	if err != nil {
		return nil, fmt.Errorf("find free port: %w", err)
	}

	// Build the opencode serve command using the full path
	args := []string{"serve", "--port", fmt.Sprintf("%d", port)}

	cmd := exec.Command(cmdPath, args...)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Env = tool_resolve.AppendExtraPaths(cmd.Env)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start agent: %w", err)
	}

	// Create reverse proxy
	targetURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Add error handler to provide better diagnostics
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
	}

	s := &agentSession{
		id:         id,
		agentID:    agentID,
		agentName:  agentDef.Name,
		projectDir: projectDir,
		port:       port,
		createdAt:  time.Now(),
		cmd:        cmd,
		proxy:      proxy,
		status:     "starting",
		done:       make(chan struct{}),
	}

	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()

	// Wait for agent server to be ready, then apply preferred model
	go func() {
		s.waitReady()
		s.mu.Lock()
		status := s.status
		s.mu.Unlock()
		if status == "running" {
			s.applyPreferredModel()
		}
	}()

	// Monitor process exit
	go func() {
		err := cmd.Wait()
		s.mu.Lock()
		if s.status != "stopped" {
			s.status = "error"
			if err != nil {
				s.err = err.Error()
			} else {
				s.err = "process exited unexpectedly"
			}
		}
		s.mu.Unlock()
		close(s.done)
	}()

	return s, nil
}

// launchCursorAdapter creates a cursor adapter session (no external process, in-process HTTP handler).
func (m *agentSessionManager) launchCursorAdapter(id string, agentDef *AgentDef, projectDir, apiKey string) (*agentSession, error) {
	adapter, err := cursor.NewAdapter(projectDir, m.settingsStore, apiKey)
	if err != nil {
		return nil, err
	}

	s := &agentSession{
		id:            id,
		agentID:       string(agentDef.ID),
		agentName:     agentDef.Name,
		projectDir:    projectDir,
		createdAt:     time.Now(),
		cursorAdapter: adapter,
		status:        "running",
		done:          make(chan struct{}),
	}

	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()

	return s, nil
}

func (s *agentSession) waitReady() {
	// Poll health endpoint
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/global/health", s.port)
	for i := 0; i < 60; i++ {
		resp, err := http.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				s.mu.Lock()
				if s.status == "starting" {
					s.status = "running"
				}
				s.mu.Unlock()
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	s.mu.Lock()
	if s.status == "starting" {
		s.status = "error"
		s.err = "agent server did not become ready within 30s"
	}
	s.mu.Unlock()
}

// preferredModelSubstring is the preferred model to auto-select when available.
const preferredModelSubstring = "kimi-k2.5"

// applyPreferredModel checks available models and sets the preferred one if found.
// First tries to apply the saved model from settings, then falls back to preferred model.
func (s *agentSession) applyPreferredModel() {
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", s.port)

	// Fetch current config to check current model
	configResp, err := http.Get(baseURL + "/config")
	if err != nil {
		return
	}
	defer configResp.Body.Close()

	var config struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(configResp.Body).Decode(&config); err != nil {
		return
	}

	// If a model is already set, don't override
	if config.Model != "" {
		return
	}

	// Try to apply the saved model from settings first
	savedModel := opencode_exposed.GetModel()
	if savedModel != "" {
		body := fmt.Sprintf(`{"model":"%s"}`, savedModel)
		req, err := http.NewRequest("PATCH", baseURL+"/config", strings.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close()
		return
	}

	// Fetch providers to find available models
	providersResp, err := http.Get(baseURL + "/config/providers")
	if err != nil {
		return
	}
	defer providersResp.Body.Close()

	var providers struct {
		Providers []struct {
			ID     string                     `json:"id"`
			Models map[string]json.RawMessage `json:"models"`
		} `json:"providers"`
	}
	if err := json.NewDecoder(providersResp.Body).Decode(&providers); err != nil {
		return
	}

	// Find a model matching the preferred substring
	for _, p := range providers.Providers {
		for modelID := range p.Models {
			if !strings.Contains(modelID, preferredModelSubstring) {
				continue
			}
			// Found preferred model, apply it
			body := fmt.Sprintf(`{"model":"%s"}`, modelID)
			req, err := http.NewRequest("PATCH", baseURL+"/config", strings.NewReader(body))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return
			}
			resp.Body.Close()
			return
		}
	}
}

func (m *agentSessionManager) get(id string) *agentSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[id]
}

func (m *agentSessionManager) list() []AgentSessionInfo {
	return m.listPaginated(1, 1000).Sessions // default to high limit for backward compatibility
}

func (m *agentSessionManager) listPaginated(page, pageSize int) *AgentSessionsResponse {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert sessions to slice for sorting
	sessionList := make([]*agentSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessionList = append(sessionList, s)
	}

	// Sort by creation time (newest first)
	sort.Slice(sessionList, func(i, j int) bool {
		return sessionList[j].createdAt.Before(sessionList[i].createdAt)
	})

	total := len(sessionList)
	totalPages := (total + pageSize - 1) / pageSize

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var pagedSessions []*agentSession
	if start < total {
		pagedSessions = sessionList[start:end]
	}

	// Convert to response format
	sessions := make([]AgentSessionInfo, 0, len(pagedSessions))
	for _, s := range pagedSessions {
		s.mu.Lock()
		info := AgentSessionInfo{
			ID:         s.id,
			AgentID:    s.agentID,
			AgentName:  s.agentName,
			ProjectDir: s.projectDir,
			Port:       s.port,
			CreatedAt:  s.createdAt.Format(time.RFC3339),
			Status:     s.status,
			Error:      s.err,
		}
		s.mu.Unlock()
		sessions = append(sessions, info)
	}

	return &AgentSessionsResponse{
		Sessions:   sessions,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

func (m *agentSessionManager) stop(id string) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()

	if !ok {
		return
	}

	s.mu.Lock()
	s.status = "stopped"
	s.mu.Unlock()

	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
}

func (s *agentSession) info() AgentSessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return AgentSessionInfo{
		ID:         s.id,
		AgentID:    s.agentID,
		AgentName:  s.agentName,
		ProjectDir: s.projectDir,
		Port:       s.port,
		CreatedAt:  s.createdAt.Format(time.RFC3339),
		Status:     s.status,
		Error:      s.err,
	}
}

// ------ HTTP Handlers ------

func handleListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agents := make([]AgentDef, len(agentDefs))
	copy(agents, agentDefs)

	for i := range agents {
		agents[i].Installed = isAgentInstalled(agents[i].ID, agents[i].Command)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

// isAgentInstalled checks if an agent is installed, considering custom binary paths
func isAgentInstalled(agentID AgentID, defaultCommand string) bool {
	// Check for custom binary path first
	customPath := GetAgentBinaryPath(agentID)
	if agentID == AgentIDOpenCode {
		customPath = getOpencodeBinaryPath()
	}
	if customPath != "" {
		_, err := tool_resolve.LookPath(customPath)
		return err == nil
	}
	// Fall back to default command
	return tool_resolve.IsAvailable(defaultCommand)
}

func getOpencodeBinaryPath() string {
	settings, err := opencode_exposed.LoadSettings()
	if err == nil && settings != nil {
		if path := strings.TrimSpace(settings.BinaryPath); path != "" {
			return path
		}
	}
	// Backward compatibility: if not migrated yet, still honor legacy agent config.
	return GetAgentBinaryPath(AgentIDOpenCode)
}

// getAgentBinaryPath returns the binary path to use for an agent
func getAgentBinaryPath(agentID AgentID, defaultCommand string) (string, error) {
	// Check for custom binary path first
	customPath := GetAgentBinaryPath(agentID)
	if agentID == AgentIDOpenCode {
		customPath = getOpencodeBinaryPath()
	}
	if customPath != "" {
		return tool_resolve.LookPath(customPath)
	}
	// Fall back to default command
	return tool_resolve.LookPath(defaultCommand)
}

// handleOpencodeAuth returns the OpenCode authentication status
func handleOpencodeAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status, err := opencode_exposed.GetAuthStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleOpencodeSettings handles GET/POST for opencode web server settings
func handleOpencodeSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := opencode_exposed.LoadSettings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)

	case http.MethodPost:
		var req opencode_exposed.Settings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := opencode_exposed.SaveSettings(&req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleOpencodeWebStatus returns the OpenCode web server status
func handleOpencodeWebStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status, err := opencode_exposed.GetWebServerStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleOpencodeServer returns info about the opencode server (port, running status)
// This is used for external session chat
func handleOpencodeServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get or start the opencode server
	server, err := opencode_internal.GetOrStartOpencodeServer()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"port":    server.Port,
		"running": server.Cmd != nil && server.Cmd.Process != nil,
	})
}

// handleOpencodeWebServerStart handles starting the web server
func handleOpencodeWebServerStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	acceptHeader := r.Header.Get("Accept")
	if acceptHeader == "text/event-stream" {
		handleOpencodeWebServerStartStreaming(w, r)
		return
	}

	resp, err := opencode_exposed.StartWebServer()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleOpencodeWebServerStartStreaming handles starting the web server with SSE streaming
func handleOpencodeWebServerStartStreaming(w http.ResponseWriter, r *http.Request) {
	sseWriter := sse.NewWriter(w)
	if sseWriter == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sseWriter.SendLog("Starting OpenCode web server...")
	resp, err := opencode_exposed.StartWebServer()
	if err != nil {
		sseWriter.SendError(fmt.Sprintf("Failed to start web server: %v", err))
		sseWriter.SendDone(map[string]string{"success": "false", "message": err.Error()})
		return
	}
	if resp == nil || !resp.Success {
		message := "Failed to start web server"
		if resp != nil && resp.Message != "" {
			message = resp.Message
		}
		sseWriter.SendError(message)
		sseWriter.SendDone(map[string]string{"success": "false", "message": message, "running": "false"})
		return
	}

	status, statusErr := opencode_exposed.GetWebServerStatus()
	port := opencode_exposed.GetWebServerPort()
	if statusErr == nil && status != nil && status.Port > 0 {
		port = status.Port
	}

	sseWriter.SendStatus("running", map[string]string{
		"port":    fmt.Sprintf("%d", port),
		"message": "Web server is running",
	})
	sseWriter.SendLog(fmt.Sprintf("✓ Web server started successfully on port %d", port))
	sseWriter.SendDone(map[string]string{"success": "true", "message": "Web server started successfully", "running": "true", "port": fmt.Sprintf("%d", port)})
}

// handleOpencodeWebServerStop handles stopping the web server
func handleOpencodeWebServerStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	acceptHeader := r.Header.Get("Accept")
	if acceptHeader == "text/event-stream" {
		handleOpencodeWebServerStopStreaming(w, r)
		return
	}

	resp, err := opencode_exposed.StopWebServer()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleOpencodeWebServerStopStreaming handles stopping the web server with SSE streaming
func handleOpencodeWebServerStopStreaming(w http.ResponseWriter, r *http.Request) {
	sseWriter := sse.NewWriter(w)
	if sseWriter == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	settings, err := opencode_exposed.LoadSettings()
	if err != nil {
		sseWriter.SendError(fmt.Sprintf("Failed to load settings: %v", err))
		sseWriter.SendDone(map[string]string{"success": "false", "message": err.Error()})
		return
	}

	if !opencode_exposed.IsWebServerRunning(settings.WebServer.Port) {
		sseWriter.SendStatus("stopped", map[string]string{"message": "Web server is already stopped"})
		sseWriter.SendLog("Web server is already stopped")
		sseWriter.SendDone(map[string]string{"success": "true", "message": "Web server is already stopped", "running": "false"})
		return
	}

	sseWriter.SendLog("Stopping OpenCode web server...")
	sseWriter.SendStatus("stopping", map[string]string{"port": fmt.Sprintf("%d", settings.WebServer.Port)})

	stopResp, err := opencode_exposed.StopWebServer()
	if err != nil {
		sseWriter.SendError(fmt.Sprintf("Failed to stop web server: %v", err))
		sseWriter.SendDone(map[string]string{"success": "false", "message": err.Error()})
		return
	}

	running := false
	message := "Web server stopped successfully"
	if stopResp != nil {
		running = stopResp.Running
		if stopResp.Message != "" {
			message = stopResp.Message
		}
	}

	if !running {
		sseWriter.SendStatus("stopped", map[string]string{
			"port":    fmt.Sprintf("%d", settings.WebServer.Port),
			"message": message,
		})
		sseWriter.SendLog(fmt.Sprintf("✓ %s", message))
		sseWriter.SendDone(map[string]string{"success": "true", "message": message, "running": "false"})
	} else {
		sseWriter.SendStatus("failed", map[string]string{
			"port":    fmt.Sprintf("%d", settings.WebServer.Port),
			"message": message,
		})
		sseWriter.SendError(message)
		sseWriter.SendDone(map[string]string{"success": "false", "message": message, "running": "true"})
	}
}

// handleOpencodeWebServerDomainMap handles domain mapping via Cloudflare
func handleOpencodeWebServerDomainMap(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req opencode_exposed.MapDomainRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Allow empty body - use defaults
			req = opencode_exposed.MapDomainRequest{}
		}

		resp, err := opencode_exposed.MapDomainViaCloudflare(req.Provider)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

	case http.MethodDelete:
		resp, err := opencode_exposed.UnmapDomain()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleOpencodeWebServerDomainMapStreaming handles domain mapping with SSE streaming support
// Supports reconnection via session_id query parameter
func handleOpencodeWebServerDomainMapStreaming(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for existing session (reconnection)
	sessionID := r.URL.Query().Get("session_id")
	startIndex := 0
	if idxStr := r.URL.Query().Get("log_index"); idxStr != "" {
		if idx, err := strconv.Atoi(idxStr); err == nil {
			startIndex = idx
		}
	}

	// For POST requests, start a new mapping operation
	var provider string
	if r.Method == http.MethodPost {
		var req opencode_exposed.MapDomainRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			req = opencode_exposed.MapDomainRequest{}
		}
		provider = req.Provider
	}

	// Start or get existing session
	session, err := opencode_exposed.MapDomainViaCloudflareStreaming(provider, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Setup SSE
	sseWriter := sse.NewWriter(w)
	if sseWriter == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send session ID for reconnection
	sseWriter.Send(map[string]string{
		"type":       "session",
		"session_id": session.ID,
	})

	// If this is a reconnection and session is already done, send completion
	if session.IsDone() {
		sseWriter.SendStatus(session.Status, map[string]string{
			"public_url": session.PublicURL,
		})
		if session.Success {
			sseWriter.SendDone(map[string]string{
				"success":    "true",
				"message":    "Domain mapping completed",
				"public_url": session.PublicURL,
			})
		} else {
			sseWriter.SendDone(map[string]string{
				"success": "false",
				"message": session.Error,
			})
		}
		return
	}

	// Send any logs that were already generated (for reconnection)
	if startIndex > 0 {
		existingLogs := session.GetLogsSince(0)
		for _, log := range existingLogs {
			if log.IsError {
				sseWriter.SendError(log.Message)
			} else {
				sseWriter.SendLog(log.Message)
			}
		}
	}

	// Stream logs until completion
	currentIndex := len(session.Logs)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			// Client disconnected - they can reconnect with session_id
			return

		case <-session.WaitDone():
			// Send any remaining logs
			logs := session.GetLogsSince(currentIndex)
			for _, log := range logs {
				if log.IsError {
					sseWriter.SendError(log.Message)
				} else {
					sseWriter.SendLog(log.Message)
				}
			}

			// Send final status
			sseWriter.SendStatus(session.Status, map[string]string{
				"public_url": session.PublicURL,
			})

			if session.Success {
				sseWriter.SendDone(map[string]string{
					"success":    "true",
					"message":    "Domain mapping completed successfully",
					"public_url": session.PublicURL,
				})
			} else {
				sseWriter.SendDone(map[string]string{
					"success": "false",
					"message": session.Error,
				})
			}
			return

		case <-ticker.C:
			// Send new logs
			logs := session.GetLogsSince(currentIndex)
			for _, log := range logs {
				if log.IsError {
					sseWriter.SendError(log.Message)
				} else {
					sseWriter.SendLog(log.Message)
				}
			}
			currentIndex = len(session.Logs)
		}
	}
}

// handleAgentEffectivePath returns the effective binary path for an agent
func handleAgentEffectivePath(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agentID := r.URL.Query().Get("agent_id")
	if agentID == "" {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
		return
	}

	// Find the agent definition to get the default command
	var defaultCommand string
	aid := AgentID(agentID)
	for _, def := range agentDefs {
		if def.ID == aid {
			defaultCommand = def.Command
			break
		}
	}
	if defaultCommand == "" {
		http.Error(w, "unknown agent", http.StatusNotFound)
		return
	}

	// Get the effective path
	effectivePath, err := getAgentBinaryPath(aid, defaultCommand)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"effective_path": effectivePath,
		"found":          err == nil,
		"error": func() string {
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
	})
}

// handleAgentConfig handles GET/POST for agent configuration
func handleAgentConfig(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent_id")

	switch r.Method {
	case http.MethodGet:
		// Return config for a specific agent or all agents
		cfg, err := LoadConfig()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if agentID != "" {
			agentCfg := cfg.Agents[agentID]
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(agentCfg)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(cfg)
		}

	case http.MethodPost:
		// Update config for a specific agent
		if agentID == "" {
			http.Error(w, "agent_id is required", http.StatusBadRequest)
			return
		}
		var req struct {
			BinaryPath string `json:"binary_path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := SetAgentBinaryPath(AgentID(agentID), req.BinaryPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleExternalSessions returns sessions from external opencode servers (CLI/web)
func handleExternalSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 5 // default page size

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	// Get or start the opencode server
	server, err := opencode_internal.GetOrStartOpencodeServer()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch sessions from opencode server
	url := fmt.Sprintf("http://127.0.0.1:%d/session", server.Port)
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Accept both 200 (no auth) and 401 (requires auth) as valid responses
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		http.Error(w, fmt.Sprintf("opencode server returned status %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}

	// If 401, return empty sessions (user needs to authenticate)
	if resp.StatusCode == http.StatusUnauthorized {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items":       []interface{}{},
			"page":        page,
			"page_size":   pageSize,
			"total":       0,
			"total_pages": 0,
			"port":        server.Port,
			"auth":        true,
		})
		return
	}

	var allSessions []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&allSessions); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply pagination
	total := len(allSessions)
	totalPages := (total + pageSize - 1) / pageSize

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var pagedSessions []map[string]interface{}
	if start < total {
		pagedSessions = allSessions[start:end]
	} else {
		pagedSessions = []map[string]interface{}{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":       pagedSessions,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
		"port":        server.Port,
	})
}

func handleAgentSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Parse pagination parameters
		page := 1
		pageSize := 10 // default page size

		if p := r.URL.Query().Get("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
				page = parsed
			}
		}
		if ps := r.URL.Query().Get("page_size"); ps != "" {
			if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
				pageSize = parsed
			}
		}

		sessions := sessionMgr.listPaginated(page, pageSize)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)

	case http.MethodPost:
		var req struct {
			AgentID    string `json:"agent_id"`
			ProjectDir string `json:"project_dir"`
			APIKey     string `json:"api_key,omitempty"` // Optional API key for cursor-agent
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		s, err := sessionMgr.launch(req.AgentID, req.ProjectDir, req.APIKey)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.info())

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		sessionMgr.stop(id)
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleExternalSessionProxy proxies requests to an external opencode server for external sessions.
func handleExternalSessionProxy(w http.ResponseWriter, r *http.Request, parts []string) {
	server, err := opencode_internal.GetOrStartOpencodeServer()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	restPath := "/"
	if len(parts) >= 3 {
		restPath = "/" + parts[2]
	}

	r.URL.Path = restPath
	r.URL.RawPath = ""

	if strings.Contains(restPath, "/message") && r.Method == http.MethodGet {
		opencode_exposed.ProxyMessages(w, r, server.Port)
		return
	}

	if restPath == "/event" || restPath == "/global/event" {
		opencode_exposed.ProxySSE(w, r, server.Port)
		return
	}

	targetURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d%s", server.Port, restPath))
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ServeHTTP(w, r)
}

// handleAgentSessionProxy proxies requests to the agent's opencode server.
// URL format: /api/agents/sessions/{sessionID}/proxy/{rest...}
func handleAgentSessionProxy(w http.ResponseWriter, r *http.Request) {
	// Parse path: /api/agents/sessions/{sessionID}/proxy/{rest}
	const prefix = "/api/agents/sessions/"
	path := strings.TrimPrefix(r.URL.Path, prefix)
	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 || parts[1] != "proxy" {
		http.NotFound(w, r)
		return
	}

	sessionID := parts[0]

	// Handle external sessions (e.g., /api/agents/sessions/external/proxy/...)
	if sessionID == "external" {
		handleExternalSessionProxy(w, r, parts)
		return
	}

	s := sessionMgr.get(sessionID)
	if s == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// Check session status before proxying
	s.mu.Lock()
	status := s.status
	errMsg := s.err
	s.mu.Unlock()

	if status == "starting" {
		http.Error(w, "session is still starting", http.StatusServiceUnavailable)
		return
	}
	if status == "error" || status == "stopped" {
		http.Error(w, fmt.Sprintf("session is not running: %s", errMsg), http.StatusServiceUnavailable)
		return
	}

	// Check if this is an SSE request (for /event endpoint)
	restPath := "/"
	if len(parts) >= 3 {
		restPath = "/" + parts[2]
	}

	// Rewrite the request path for the proxy target
	r.URL.Path = restPath
	r.URL.RawPath = ""

	// If this session uses the cursor adapter, route to it
	if s.cursorAdapter != nil {
		s.cursorAdapter.ServeHTTP(w, r)
		return
	}

	// For config PATCH, transform model from object to string for opencode
	if restPath == "/config" && r.Method == http.MethodPatch {
		opencode_exposed.ProxyConfigUpdate(w, r, s.port)
		return
	}

	// For SSE endpoints, convert OpenCode events to ACP
	if restPath == "/event" || restPath == "/global/event" {
		opencode_exposed.ProxySSE(w, r, s.port)
		return
	}

	// For message endpoints, convert response to ACP format
	if strings.Contains(restPath, "/message") && r.Method == http.MethodGet {
		opencode_exposed.ProxyMessages(w, r, s.port)
		return
	}

	s.proxy.ServeHTTP(w, r)
}
