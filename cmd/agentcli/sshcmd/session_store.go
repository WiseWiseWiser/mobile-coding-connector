package sshcmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

// FileSessionStore persists Session JSON under {Root}/ssh-sessions/{profileID}.json.
type FileSessionStore struct {
	Root string
}

// sessionWire is the on-disk snake_case JSON shape for Session.
type sessionWire struct {
	LocalPort int    `json:"local_port"`
	User      string `json:"user"`
	ConfigDir string `json:"config_dir"`
	ServePID  int    `json:"serve_pid"`
	ProfileID string `json:"profile_id"`
	Alive     bool   `json:"alive"`
}

func (s *FileSessionStore) sessionPath(profileID string) string {
	return filepath.Join(s.Root, "ssh-sessions", profileID+".json")
}

// Load reads a session file. Missing file → (nil, nil). Corrupt JSON → error.
// When Alive is true and ServePID is non-zero but the process is dead, Alive is forced false.
func (s *FileSessionStore) Load(profileID string) (*Session, error) {
	if s == nil {
		return nil, errors.New("FileSessionStore is nil")
	}
	path := s.sessionPath(profileID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var wire sessionWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return nil, err
	}
	sess := &Session{
		LocalPort: wire.LocalPort,
		User:      wire.User,
		ConfigDir: wire.ConfigDir,
		ServePID:  wire.ServePID,
		ProfileID: wire.ProfileID,
		Alive:     wire.Alive,
	}
	// ServePID == 0: skip process liveness check.
	if sess.Alive && sess.ServePID != 0 && !processAlive(sess.ServePID) {
		sess.Alive = false
	}
	return sess, nil
}

// Save writes session JSON (snake_case keys) under the profile path.
func (s *FileSessionStore) Save(sess *Session) error {
	if s == nil {
		return errors.New("FileSessionStore is nil")
	}
	if sess == nil {
		return errors.New("session is nil")
	}
	dir := filepath.Join(s.Root, "ssh-sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	wire := sessionWire{
		LocalPort: sess.LocalPort,
		User:      sess.User,
		ConfigDir: sess.ConfigDir,
		ServePID:  sess.ServePID,
		ProfileID: sess.ProfileID,
		Alive:     sess.Alive,
	}
	data, err := json.MarshalIndent(wire, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := s.sessionPath(sess.ProfileID)
	return os.WriteFile(path, data, 0o644)
}

// Clear removes the session file for profileID. Missing file is not an error.
func (s *FileSessionStore) Clear(profileID string) error {
	if s == nil {
		return errors.New("FileSessionStore is nil")
	}
	path := s.sessionPath(profileID)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// processAlive reports whether pid refers to a living process.
// Uses kill(pid, 0): nil or EPERM means the process exists.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	if err == syscall.EPERM {
		return true
	}
	return false
}
