package agents

import "testing"

// Doctest-exported helpers for external harness packages (import github.com/.../server/agents).

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