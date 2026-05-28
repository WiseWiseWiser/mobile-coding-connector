package custom

import (
	"path/filepath"

	opencode_sessions "github.com/xhd2015/agent-traces/agent/opencode/ai_critic/sessions"
)

const SessionsDirName = "sessions"

type SessionData = opencode_sessions.SessionData

func SessionsDir(agentID string) string {
	agentDir := AgentDir(agentID)
	if agentDir == "" {
		return ""
	}
	return filepath.Join(agentDir, SessionsDirName)
}

func SessionDir(agentID, sessionID string) string {
	sessionsDir := SessionsDir(agentID)
	if sessionsDir == "" {
		return ""
	}
	return filepath.Join(sessionsDir, sessionID)
}

func SessionDataPath(agentID, sessionID string) string {
	sessionDir := SessionDir(agentID, sessionID)
	if sessionDir == "" {
		return ""
	}
	return filepath.Join(sessionDir, "session.json")
}

func SaveSession(session *SessionData) error {
	return opencode_sessions.Save(SessionsDir(session.AgentID), session)
}

func LoadSession(agentID, sessionID string) (*SessionData, error) {
	dir := SessionsDir(agentID)
	if dir == "" {
		return nil, nil
	}
	return opencode_sessions.Load(dir, sessionID)
}

func ListSessions(agentID string) ([]SessionData, error) {
	dir := SessionsDir(agentID)
	if dir == "" {
		return nil, nil
	}
	return opencode_sessions.List(dir)
}

func UpdateSessionStatus(agentID, sessionID, status, errMsg string) error {
	dir := SessionsDir(agentID)
	if dir == "" {
		return nil
	}
	return opencode_sessions.UpdateStatus(dir, sessionID, status, errMsg)
}

func DeleteSession(agentID, sessionID string) error {
	dir := SessionsDir(agentID)
	if dir == "" {
		return nil
	}
	return opencode_sessions.Delete(dir, sessionID)
}
