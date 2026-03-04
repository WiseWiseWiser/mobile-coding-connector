package agents

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/custom"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

type LaunchCustomAgentResult struct {
	SessionID string
	Port      int
	URL       string
}

func LaunchCustomAgent(agentID string, projectDir string) (*LaunchCustomAgentResult, error) {
	agent, err := custom.LoadAgent(agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("agent not found: %s", agentID)
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
	sessionID := fmt.Sprintf("%s-%d", agentID, now.UnixMilli())

	sessionMgr.mu.Lock()
	sessionMgr.counter++
	sessionMgr.mu.Unlock()

	session := &customAgentSession{
		id:         sessionID,
		agentID:    agentID,
		projectDir: projectDir,
		port:       port,
		createdAt:  now,
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
		CreatedAt:  now.Format(time.RFC3339),
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
			AgentName:  "Custom: " + s.agentID,
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
