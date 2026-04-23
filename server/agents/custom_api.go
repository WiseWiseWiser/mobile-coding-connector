package agents

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/custom"
	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/opencode/common_opencode"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

type LaunchCustomAgentResult struct {
	SessionID string
	Port      int
	URL       string
}

func LaunchCustomAgent(agentID string, projectDir string, resumeSessionID string) (*LaunchCustomAgentResult, error) {
	agent, err := custom.LoadAgent(agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	var (
		sessionID string
		createdAt time.Time
	)
	if resumeSessionID != "" {
		if running := GetRunningCustomAgentSession(resumeSessionID); running != nil {
			if running.AgentID != agentID {
				return nil, fmt.Errorf("session %s belongs to agent %s, not %s", resumeSessionID, running.AgentID, agentID)
			}
			if projectDir != "" && running.ProjectDir != "" && running.ProjectDir != projectDir {
				return nil, fmt.Errorf("session %s belongs to project %s, not %s", resumeSessionID, running.ProjectDir, projectDir)
			}
			return &LaunchCustomAgentResult{
				SessionID: running.ID,
				Port:      running.Port,
				URL:       fmt.Sprintf("http://127.0.0.1:%d", running.Port),
			}, nil
		}

		savedSession, err := custom.LoadSession(agentID, resumeSessionID)
		if err != nil {
			return nil, fmt.Errorf("load session %s: %w", resumeSessionID, err)
		}
		if savedSession == nil {
			return nil, fmt.Errorf("session not found: %s", resumeSessionID)
		}
		if projectDir == "" {
			projectDir = savedSession.ProjectDir
		} else if savedSession.ProjectDir != "" && savedSession.ProjectDir != projectDir {
			return nil, fmt.Errorf("session %s belongs to project %s, not %s", resumeSessionID, savedSession.ProjectDir, projectDir)
		}
		sessionID = savedSession.ID
		if t, err := time.Parse(time.RFC3339, savedSession.CreatedAt); err == nil {
			createdAt = t
		}
	}

	if info, err := os.Stat(projectDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("invalid project directory: %s", projectDir)
	}

	if err := custom.GenerateOpencodeConfig(agentID); err != nil {
		return nil, fmt.Errorf("failed to generate config: %w", err)
	}

	configDir := custom.GetOpencodeConfigDir(agentID)

	port, err := findFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find free port: %w", err)
	}

	cmd, err := tool_exec.New("opencode", []string{"serve", "--port", fmt.Sprintf("%d", port)}, &tool_exec.Options{
		Dir: projectDir,
		Env: map[string]string{
			"OPENCODE_CONFIG_DIR": configDir,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create command: %w", err)
	}

	cmd.Cmd.Env = append(cmd.Cmd.Env, "TERM=xterm-256color")
	cmd.Cmd.Env = tool_resolve.AppendExtraPaths(cmd.Cmd.Env)
	cmd.Cmd.Stdout = os.Stdout
	cmd.Cmd.Stderr = os.Stderr

	if err := cmd.Cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start agent: %w", err)
	}

	now := time.Now()
	if createdAt.IsZero() {
		createdAt = now
	}
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s-%d", agentID, now.UnixMilli())
	}

	sessionMgr.mu.Lock()
	sessionMgr.counter++
	sessionMgr.mu.Unlock()

	session := &customAgentSession{
		id:         sessionID,
		agentID:    agentID,
		agentName:  agent.Name,
		projectDir: projectDir,
		port:       port,
		createdAt:  createdAt,
		cmd:        cmd.Cmd,
	}

	sessionsMu.Lock()
	if customAgentSessions == nil {
		customAgentSessions = make(map[string]*customAgentSession)
	}
	customAgentSessions[sessionID] = session
	sessionsMu.Unlock()

	sessionData := &custom.SessionData{
		ID:         sessionID,
		AgentID:    agentID,
		AgentName:  agent.Name,
		ProjectDir: projectDir,
		Port:       port,
		CreatedAt:  createdAt.Format(time.RFC3339),
		Status:     "running",
	}
	custom.SaveSession(sessionData)

	go monitorCustomAgentProcess(session)

	return &LaunchCustomAgentResult{
		SessionID: sessionID,
		Port:      port,
		URL:       fmt.Sprintf("http://127.0.0.1:%d", port),
	}, nil
}

func monitorCustomAgentProcess(session *customAgentSession) {
	if session.cmd == nil {
		return
	}
	err := session.cmd.Wait()

	status := "stopped"
	errMsg := ""
	if err != nil {
		status = "error"
		errMsg = err.Error()
	}

	sessionsMu.Lock()
	delete(customAgentSessions, session.id)
	sessionsMu.Unlock()

	custom.UpdateSessionStatus(session.agentID, session.id, status, errMsg)
}

type customAgentSession struct {
	id         string
	agentID    string
	agentName  string
	projectDir string
	port       int
	createdAt  time.Time
	cmd        *exec.Cmd
}

var (
	customAgentSessions map[string]*customAgentSession
	sessionsMu          sync.Mutex
)

func StopCustomAgentSession(sessionID string) error {
	sessionsMu.Lock()
	session, ok := customAgentSessions[sessionID]
	if ok {
		delete(customAgentSessions, sessionID)
	}
	sessionsMu.Unlock()

	if !ok {
		return fmt.Errorf("session not found")
	}

	if session.cmd != nil && session.cmd.Process != nil {
		session.cmd.Process.Kill()
	}

	custom.UpdateSessionStatus(session.agentID, sessionID, "stopped", "")
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
			AgentName:  s.agentName,
			ProjectDir: s.projectDir,
			Port:       s.port,
			CreatedAt:  s.createdAt.Format(time.RFC3339),
			Status:     "running",
		})
	}

	return result
}

// GetRunningCustomSessionIDs returns the set of currently running custom agent session IDs.
func GetRunningCustomSessionIDs() map[string]bool {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	ids := make(map[string]bool, len(customAgentSessions))
	for id := range customAgentSessions {
		ids[id] = true
	}
	return ids
}

// GetRunningCustomAgentSession returns info about a currently running custom-agent session.
func GetRunningCustomAgentSession(sessionID string) *AgentSessionInfo {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()

	session := customAgentSessions[sessionID]
	if session == nil {
		return nil
	}

	return &AgentSessionInfo{
		ID:         session.id,
		AgentID:    session.agentID,
		AgentName:  session.agentName,
		ProjectDir: session.projectDir,
		Port:       session.port,
		CreatedAt:  session.createdAt.Format(time.RFC3339),
		Status:     "running",
	}
}

func WaitForCustomAgentSessionReady(sessionID string, timeout time.Duration) error {
	session := GetRunningCustomAgentSession(sessionID)
	if session == nil {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	return common_opencode.WaitForSessionReady(session.Port, timeout)
}
