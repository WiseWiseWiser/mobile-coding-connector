package opencode_serve_children

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

const (
	KindHeadlessAgent = "headless-agent"
	KindCustomAgent   = "custom-agent"
)

type ChildEntry struct {
	Kind       string `json:"kind"`
	SessionID  string `json:"session_id"`
	PID        int    `json:"pid"`
	Port       int    `json:"port"`
	ProjectDir string `json:"project_dir"`
	AgentID    string `json:"agent_id"`
	StartedAt  string `json:"started_at"`
}

type Registry struct {
	Children []ChildEntry `json:"children"`
}

// ResolveDataDir returns configHome when set, else AI_CRITIC_HOME, else config.DataDir.
func ResolveDataDir(configHome string) string {
	if configHome != "" {
		return configHome
	}
	if dir := os.Getenv("AI_CRITIC_HOME"); dir != "" {
		return dir
	}
	return config.DataDir
}

func registryPath(configHome string) string {
	return filepath.Join(ResolveDataDir(configHome), "opencode-serve-children.json")
}

func lockPath(configHome string) string {
	return filepath.Join(ResolveDataDir(configHome), "opencode-serve-children.lock")
}

func ensureDataDir(configHome string) error {
	return os.MkdirAll(ResolveDataDir(configHome), 0755)
}

func WithLock(configHome string, fn func() error) error {
	if err := ensureDataDir(configHome); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	lockFile := lockPath(configHome)
	fd, err := syscall.Open(lockFile, syscall.O_CREAT|syscall.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open lock file: %w", err)
	}
	defer syscall.Close(fd)

	if err := syscall.Flock(fd, syscall.LOCK_EX); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)

	return fn()
}

func Load(configHome string) (*Registry, error) {
	data, err := os.ReadFile(registryPath(configHome))
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{Children: nil}, nil
		}
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	if len(data) == 0 {
		return &Registry{Children: nil}, nil
	}

	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}
	return &reg, nil
}

func saveRegistry(configHome string, reg *Registry) error {
	if err := ensureDataDir(configHome); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(registryPath(configHome), data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}
	return nil
}

func Add(configHome string, entry ChildEntry) error {
	if entry.StartedAt == "" {
		entry.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return WithLock(configHome, func() error {
		reg, err := Load(configHome)
		if err != nil {
			return err
		}
		for i, child := range reg.Children {
			if child.SessionID == entry.SessionID {
				reg.Children[i] = entry
				return saveRegistry(configHome, reg)
			}
		}
		reg.Children = append(reg.Children, entry)
		return saveRegistry(configHome, reg)
	})
}

func Remove(configHome string, sessionID string) error {
	return WithLock(configHome, func() error {
		reg, err := Load(configHome)
		if err != nil {
			return err
		}
		filtered := reg.Children[:0]
		for _, child := range reg.Children {
			if child.SessionID != sessionID {
				filtered = append(filtered, child)
			}
		}
		reg.Children = filtered
		if len(reg.Children) == 0 {
			return Clear(configHome)
		}
		return saveRegistry(configHome, reg)
	})
}

func Clear(configHome string) error {
	path := registryPath(configHome)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove registry: %w", err)
	}
	return nil
}