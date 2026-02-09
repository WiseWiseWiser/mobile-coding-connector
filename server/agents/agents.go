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
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/cursor"
	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/opencode"
	"github.com/xhd2015/lifelog-private/ai-critic/server/settings"
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

func (m *agentSessionManager) launch(agentID, projectDir string) (*agentSession, error) {
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
		return m.launchCursorAdapter(id, agentDef, projectDir)
	}

	// Check command is installed and get full path
	cmdPath, err := tool_resolve.LookPath(agentDef.Command)
	if err != nil {
		return nil, fmt.Errorf("agent %s is not installed (%s not found)", agentDef.Name, agentDef.Command)
	}

	// Find a free port
	port, err := findFreePort()
	if err != nil {
		return nil, fmt.Errorf("find free port: %w", err)
	}

	// Build the opencode serve command using the full path
	cmd := exec.Command(cmdPath, "serve", "--port", fmt.Sprintf("%d", port))
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

	// Wait for agent server to be ready
	go s.waitReady()

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
func (m *agentSessionManager) launchCursorAdapter(id string, agentDef *AgentDef, projectDir string) (*agentSession, error) {
	adapter, err := cursor.NewAdapter(projectDir, m.settingsStore)
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

func (m *agentSessionManager) get(id string) *agentSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[id]
}

func (m *agentSessionManager) list() []AgentSessionInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]AgentSessionInfo, 0, len(m.sessions))
	for _, s := range m.sessions {
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
		result = append(result, info)
	}
	return result
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
		agents[i].Installed = tool_resolve.IsAvailable(agents[i].Command)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func handleAgentSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sessions := sessionMgr.list()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)

	case http.MethodPost:
		var req struct {
			AgentID    string `json:"agent_id"`
			ProjectDir string `json:"project_dir"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		s, err := sessionMgr.launch(req.AgentID, req.ProjectDir)
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
