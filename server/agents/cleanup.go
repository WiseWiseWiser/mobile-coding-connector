package agents

import (
	"github.com/xhd2015/ai-critic/server/agents/opencode_serve_children"
)

// CleanupAllOpencodeServe kills all registered opencode serve children and clears registries.
func CleanupAllOpencodeServe() error {
	sessionMgr.mu.Lock()
	for id, s := range sessionMgr.sessions {
		if s.cmd != nil && s.cmd.Process != nil {
			opencode_serve_children.KillChild(s.cmd.Process.Pid, s.port)
		}
		delete(sessionMgr.sessions, id)
	}
	sessionMgr.mu.Unlock()

	sessionsMu.Lock()
	for id, s := range customAgentSessions {
		if s.cmd != nil && s.cmd.Process != nil {
			opencode_serve_children.KillChild(s.cmd.Process.Pid, s.port)
		}
		delete(customAgentSessions, id)
	}
	sessionsMu.Unlock()

	return opencode_serve_children.CleanupAll("")
}