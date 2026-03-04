package custom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"time"
)

const SessionsDirName = "sessions"

type SessionData struct {
	ID         string `json:"id"`
	AgentID    string `json:"agent_id"`
	AgentName  string `json:"agent_name"`
	ProjectDir string `json:"project_dir"`
	Port       int    `json:"port"`
	CreatedAt  string `json:"created_at"`
	Status     string `json:"status"` // "starting", "running", "stopped", "error"
	Error      string `json:"error,omitempty"`
}

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
	sessionDir := SessionDir(session.AgentID, session.ID)
	if sessionDir == "" {
		return nil
	}
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(SessionDataPath(session.AgentID, session.ID), data, 0644)
}

func LoadSession(agentID, sessionID string) (*SessionData, error) {
	dataPath := SessionDataPath(agentID, sessionID)
	if dataPath == "" {
		return nil, nil
	}
	data, err := os.ReadFile(dataPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func ListSessions(agentID string) ([]SessionData, error) {
	sessionsDir := SessionsDir(agentID)
	if sessionsDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(sessionsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var sessions []SessionData
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		session, err := LoadSession(agentID, entry.Name())
		if err != nil || session == nil {
			continue
		}
		sessions = append(sessions, *session)
	}

	slices.SortFunc(sessions, func(a, b SessionData) int {
		ta, _ := time.Parse(time.RFC3339, a.CreatedAt)
		tb, _ := time.Parse(time.RFC3339, b.CreatedAt)
		if tb.Before(ta) {
			return -1
		}
		if ta.Before(tb) {
			return 1
		}
		return 0
	})
	return sessions, nil
}

func UpdateSessionStatus(agentID, sessionID, status, errMsg string) error {
	session, err := LoadSession(agentID, sessionID)
	if err != nil || session == nil {
		return err
	}
	session.Status = status
	session.Error = errMsg
	return SaveSession(session)
}

func DeleteSession(agentID, sessionID string) error {
	sessionDir := SessionDir(agentID, sessionID)
	if sessionDir == "" {
		return nil
	}
	return os.RemoveAll(sessionDir)
}
