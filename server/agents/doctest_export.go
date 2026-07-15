package agents

import (
	"testing"

	"github.com/xhd2015/ai-critic/server/agents/opencode_serve_children"
)

// Doctest-exported helpers for external harness packages (import github.com/.../server/agents).

// OpencodeServeChildEntry is exported for doctest registry assertions.
type OpencodeServeChildEntry struct {
	Kind       string
	SessionID  string
	PID        int
	Port       int
	ProjectDir string
	AgentID    string
	StartedAt  string
}

// TestExported_StripOpencodeResolutionForDoctest makes opencode/grok resolution use PATH only.
func TestExported_StripOpencodeResolutionForDoctest(t *testing.T) {
	doctestIgnoreOpencodeCustomPaths = true
	t.Cleanup(func() { doctestIgnoreOpencodeCustomPaths = false })
}

func TestExported_LaunchAgentSession(agentID, projectDir, model string) (AgentSessionInfo, error) {
	_ = model
	s, err := sessionMgr.launch(agentID, projectDir, "")
	if err != nil {
		return AgentSessionInfo{}, err
	}
	return s.info(), nil
}

func TestExported_StopAgentSession(sessionID string) {
	sessionMgr.stop(sessionID)
}

func TestExported_PreferredModelSubstringForAgent(agentID string) string {
	return PreferredModelSubstringForAgent(agentID)
}

func TestExported_ReadOpencodeServeChildrenRegistry() ([]OpencodeServeChildEntry, error) {
	reg, err := opencode_serve_children.Load("")
	if err != nil {
		return nil, err
	}
	out := make([]OpencodeServeChildEntry, 0, len(reg.Children))
	for _, child := range reg.Children {
		out = append(out, OpencodeServeChildEntry{
			Kind:       child.Kind,
			SessionID:  child.SessionID,
			PID:        child.PID,
			Port:       child.Port,
			ProjectDir: child.ProjectDir,
			AgentID:    child.AgentID,
			StartedAt:  child.StartedAt,
		})
	}
	return out, nil
}

func TestExported_CleanupAllOpencodeServe() error {
	return CleanupAllOpencodeServe()
}

func TestExported_LaunchCustomAgent(agentID, projectDir string) (LaunchCustomAgentResult, error) {
	result, err := LaunchCustomAgent(agentID, projectDir, "")
	if err != nil {
		return LaunchCustomAgentResult{}, err
	}
	return *result, nil
}

func TestExported_StopCustomAgentSession(sessionID string) {
	_ = StopCustomAgentSession(sessionID)
}