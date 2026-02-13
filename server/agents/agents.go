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
	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/opencode"
	"github.com/xhd2015/lifelog-private/ai-critic/server/settings"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
	"github.com/xhd2015/lifelog-private/ai-critic/server/subprocess"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

// AgentDef defines a supported coding agent
type AgentDef struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Command     string `json:"command"`
	// Installed is set dynamically by checking if the command is available
	Installed bool `json:"installed"`
	// Headless indicates this agent supports headless server mode
	Headless bool `json:"headless"`
}

// All supported agents - add new agents here
var agentDefs = []AgentDef{
	{
		ID:          "opencode",
		Name:        "OpenCode",
		Description: "AI-powered coding assistant with headless server mode",
		Command:     "opencode",
		Headless:    true,
	},
	{
		ID:          "claude-code",
		Name:        "Claude Code",
		Description: "Anthropic's Claude coding agent (CLI)",
		Command:     "claude",
	},
	{
		ID:          "codex",
		Name:        "Codex",
		Description: "OpenAI Codex CLI agent",
		Command:     "codex",
	},
	{
		ID:          "cursor-agent",
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
	mux.HandleFunc("/api/agents/opencode/web-server/control", handleOpencodeWebServerControl)
	mux.HandleFunc("/api/agents/opencode/web-server/domain-map", handleOpencodeWebServerDomainMap)
	mux.HandleFunc("/api/agents/opencode/web-server/domain-map/stream", handleOpencodeWebServerDomainMapStreaming)
	mux.HandleFunc("/api/agents/sessions", handleAgentSessions)
	// Proxy: /api/agents/sessions/{sessionID}/proxy/... -> opencode server
	mux.HandleFunc("/api/agents/sessions/", handleAgentSessionProxy)
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
	// Find the agent def
	var agentDef *AgentDef
	for i := range agentDefs {
		if agentDefs[i].ID == agentID {
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
	if agentDef.ID == "cursor-agent" {
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
		agentID:       agentDef.ID,
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
	savedModel := opencode.GetModel()
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
func isAgentInstalled(agentID, defaultCommand string) bool {
	// Check for custom binary path first
	customPath := GetAgentBinaryPath(agentID)
	if customPath != "" {
		_, err := tool_resolve.LookPath(customPath)
		return err == nil
	}
	// Fall back to default command
	return tool_resolve.IsAvailable(defaultCommand)
}

// getAgentBinaryPath returns the binary path to use for an agent
func getAgentBinaryPath(agentID, defaultCommand string) (string, error) {
	// Check for custom binary path first
	customPath := GetAgentBinaryPath(agentID)
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

	status, err := opencode.GetAuthStatus()
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
		settings, err := opencode.LoadSettings()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)

	case http.MethodPost:
		var req opencode.Settings
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := opencode.SaveSettings(&req); err != nil {
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

	status, err := opencode.GetWebServerStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleOpencodeWebServerControl handles start/stop operations for the web server
// Supports both JSON and SSE streaming responses based on Accept header
func handleOpencodeWebServerControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req opencode.WebServerControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Get user-configured binary path for opencode agent
	customPath := GetAgentBinaryPath("opencode")

	// Check if client wants streaming (SSE)
	acceptHeader := r.Header.Get("Accept")
	if acceptHeader == "text/event-stream" {
		handleOpencodeWebServerControlStreaming(w, r, req.Action, customPath)
		return
	}

	// Non-streaming JSON response
	resp, err := opencode.ControlWebServer(req.Action, customPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleOpencodeWebServerControlStreaming handles start/stop with SSE streaming
func handleOpencodeWebServerControlStreaming(w http.ResponseWriter, r *http.Request, action string, customPath string) {
	sseWriter := sse.NewWriter(w)
	if sseWriter == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	settings, err := opencode.LoadSettings()
	if err != nil {
		sseWriter.SendError(fmt.Sprintf("Failed to load settings: %v", err))
		sseWriter.SendDone(map[string]string{"success": "false", "message": err.Error()})
		return
	}

	switch action {
	case "start":
		// Check if already running via subprocess manager
		manager := subprocess.GetManager()
		if manager.IsRunning(opencode.WebServerProcessID) {
			sseWriter.SendLog("Web server is already running (managed)")
			sseWriter.SendDone(map[string]string{"success": "true", "message": "Web server is already running", "running": "true"})
			return
		}

		// Also check via HTTP health check
		if opencode.IsWebServerRunning(settings.WebServer.Port) {
			sseWriter.SendLog("Web server is already running")
			sseWriter.SendDone(map[string]string{"success": "true", "message": "Web server is already running", "running": "true"})
			return
		}

		sseWriter.SendLog(fmt.Sprintf("Starting OpenCode web server on port %d...", settings.WebServer.Port))

		// Start the web server using opencode command with proper environment
		cmd, err := tool_exec.New("opencode", []string{"web", "--port", fmt.Sprintf("%d", settings.WebServer.Port)}, &tool_exec.Options{
			CustomPath: customPath,
		})
		if err != nil {
			sseWriter.SendError(fmt.Sprintf("Failed to create command: %v", err))
			sseWriter.SendDone(map[string]string{"success": "false", "message": err.Error()})
			return
		}

		// Log the full command being executed
		cmdStr := cmd.Cmd.Path
		for _, arg := range cmd.Cmd.Args[1:] {
			cmdStr += " " + arg
		}
		sseWriter.SendLog(fmt.Sprintf("Executing: %s", cmdStr))

		// Health checker function
		healthChecker := func() bool {
			return opencode.IsWebServerRunning(settings.WebServer.Port)
		}

		// Start the process via subprocess manager (non-blocking)
		process, err := manager.StartProcess(opencode.WebServerProcessID, "OpenCode Web Server", cmd.Cmd, healthChecker, true)
		if err != nil {
			sseWriter.SendError(fmt.Sprintf("Failed to start web server: %v", err))
			sseWriter.SendDone(map[string]string{"success": "false", "message": err.Error()})
			return
		}

		sseWriter.SendLog("Process started, waiting for health check...")
		sseWriter.SendStatus("starting", map[string]string{"port": fmt.Sprintf("%d", settings.WebServer.Port)})

		// Wait for the server to be ready with periodic status updates
		running := false
		checkInterval := 500 * time.Millisecond
		timeout := 10 * time.Second
		deadline := time.Now().Add(timeout)
		checkCount := 0

		for time.Now().Before(deadline) {
			checkCount++
			if process.HealthChecker != nil && process.HealthChecker() {
				running = true
				break
			}

			// Send periodic status update every 2 checks (1 second)
			if checkCount%2 == 0 {
				elapsed := time.Since(process.StartTime).Seconds()
				sseWriter.SendStatus("waiting", map[string]string{
					"elapsed": fmt.Sprintf("%.1f", elapsed),
					"port":    fmt.Sprintf("%d", settings.WebServer.Port),
				})
				sseWriter.SendLog(fmt.Sprintf("Waiting for server to be ready... (%.1fs)", elapsed))
			}

			time.Sleep(checkInterval)
		}

		// Update settings
		settings.WebServer.Enabled = running
		opencode.SaveSettings(settings)

		if running {
			sseWriter.SendStatus("running", map[string]string{
				"port":    fmt.Sprintf("%d", settings.WebServer.Port),
				"message": "Web server is running",
			})
			sseWriter.SendLog(fmt.Sprintf("✓ Web server started successfully on port %d", settings.WebServer.Port))
			sseWriter.SendDone(map[string]string{"success": "true", "message": "Web server started successfully", "running": "true"})
		} else {
			sseWriter.SendStatus("failed", map[string]string{
				"port":    fmt.Sprintf("%d", settings.WebServer.Port),
				"message": "Health check timeout",
			})
			sseWriter.SendError("Web server process started but health check failed")
			sseWriter.SendDone(map[string]string{"success": "false", "message": "Web server may not be ready", "running": "false"})
		}

	case "stop":
		manager := subprocess.GetManager()

		// Check if already stopped
		if !opencode.IsWebServerRunning(settings.WebServer.Port) && !manager.IsRunning(opencode.WebServerProcessID) {
			sseWriter.SendStatus("stopped", map[string]string{"message": "Web server is already stopped"})
			sseWriter.SendLog("Web server is already stopped")
			sseWriter.SendDone(map[string]string{"success": "true", "message": "Web server is already stopped", "running": "false"})
			return
		}

		sseWriter.SendLog("Stopping OpenCode web server...")
		sseWriter.SendStatus("stopping", map[string]string{"port": fmt.Sprintf("%d", settings.WebServer.Port)})

		// First try to stop via subprocess manager
		if manager.IsRunning(opencode.WebServerProcessID) {
			sseWriter.SendLog("Stopping via subprocess manager...")
			if err := manager.StopProcess(opencode.WebServerProcessID); err != nil {
				sseWriter.SendLog(fmt.Sprintf("Subprocess manager stop failed: %v", err))
			}
		}

		// Also try the standard opencode stop command
		cmd, err := tool_exec.New("opencode", []string{"web", "stop"}, &tool_exec.Options{
			CustomPath: customPath,
		})
		if err != nil {
			sseWriter.SendStatus("error", map[string]string{"message": fmt.Sprintf("Failed to create command: %v", err)})
			sseWriter.SendError(fmt.Sprintf("Failed to create command: %v", err))
			sseWriter.SendDone(map[string]string{"success": "false", "message": err.Error()})
			return
		}

		// Log the full command being executed
		cmdStr := cmd.Cmd.Path
		for _, arg := range cmd.Cmd.Args[1:] {
			cmdStr += " " + arg
		}
		sseWriter.SendLog(fmt.Sprintf("Executing: %s", cmdStr))

		err = sseWriter.StreamCmd(cmd.Cmd)
		if err != nil {
			sseWriter.SendStatus("error", map[string]string{"message": fmt.Sprintf("Failed to stop web server: %v", err)})
			sseWriter.SendError(fmt.Sprintf("Failed to stop web server: %v", err))
			sseWriter.SendDone(map[string]string{"success": "false", "message": err.Error()})
			return
		}

		// Wait a moment and check if it's stopped
		time.Sleep(1 * time.Second)
		running := opencode.IsWebServerRunning(settings.WebServer.Port)

		// If still running, use pure Go implementation to kill the process by port
		if running {
			sseWriter.SendLog(fmt.Sprintf("Web server still running on port %d, attempting to kill process directly...", settings.WebServer.Port))
			if err := opencode.KillProcessByPort(settings.WebServer.Port); err != nil {
				sseWriter.SendLog(fmt.Sprintf("Failed to kill process by port: %v", err))
			} else {
				sseWriter.SendLog("Sent kill signal to process, waiting for shutdown...")
				// Wait another moment and check again
				time.Sleep(500 * time.Millisecond)
				running = opencode.IsWebServerRunning(settings.WebServer.Port)
			}
		}

		// Update settings
		settings.WebServer.Enabled = running
		opencode.SaveSettings(settings)

		if !running {
			sseWriter.SendStatus("stopped", map[string]string{
				"port":    fmt.Sprintf("%d", settings.WebServer.Port),
				"message": "Web server stopped successfully",
			})
			sseWriter.SendLog("✓ Web server stopped successfully")
			sseWriter.SendDone(map[string]string{"success": "true", "message": "Web server stopped successfully", "running": "false"})
		} else {
			sseWriter.SendStatus("failed", map[string]string{
				"port":    fmt.Sprintf("%d", settings.WebServer.Port),
				"message": "Server may still be running",
			})
			sseWriter.SendError("Web server stop command executed but server may still be running")
			sseWriter.SendDone(map[string]string{"success": "false", "message": "Web server may still be running", "running": "true"})
		}

	default:
		sseWriter.SendError(fmt.Sprintf("Invalid action: %s", action))
		sseWriter.SendDone(map[string]string{"success": "false", "message": fmt.Sprintf("Invalid action: %s", action)})
	}
}

// handleOpencodeWebServerDomainMap handles domain mapping via Cloudflare
func handleOpencodeWebServerDomainMap(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var req opencode.MapDomainRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Allow empty body - use defaults
			req = opencode.MapDomainRequest{}
		}

		resp, err := opencode.MapDomainViaCloudflare(req.Provider)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

	case http.MethodDelete:
		resp, err := opencode.UnmapDomain()
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
		var req opencode.MapDomainRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			req = opencode.MapDomainRequest{}
		}
		provider = req.Provider
	}

	// Start or get existing session
	session, err := opencode.MapDomainViaCloudflareStreaming(provider, sessionID)
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
	for _, def := range agentDefs {
		if def.ID == agentID {
			defaultCommand = def.Command
			break
		}
	}
	if defaultCommand == "" {
		http.Error(w, "unknown agent", http.StatusNotFound)
		return
	}

	// Get the effective path
	effectivePath, err := getAgentBinaryPath(agentID, defaultCommand)

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
		if err := SetAgentBinaryPath(agentID, req.BinaryPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleAgentSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Parse pagination parameters
		page := 1
		pageSize := 20 // default page size

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
		opencode.ProxyConfigUpdate(w, r, s.port)
		return
	}

	// For SSE endpoints, convert OpenCode events to ACP
	if restPath == "/event" || restPath == "/global/event" {
		opencode.ProxySSE(w, r, s.port)
		return
	}

	// For message endpoints, convert response to ACP format
	if strings.Contains(restPath, "/message") && r.Method == http.MethodGet {
		opencode.ProxyMessages(w, r, s.port)
		return
	}

	s.proxy.ServeHTTP(w, r)
}
