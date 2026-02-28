package internal_opencode

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

type InternalServerInfo struct {
	PID       int   `json:"pid"`
	Port      int   `json:"port"`
	StartTime int64 `json:"start_time"`
}

func registryPath() string {
	return config.OpencodeInternalServerRegistry
}

func lockPath() string {
	return config.OpencodeInternalServerLock
}

func ensureDataDir() error {
	return os.MkdirAll(config.DataDir, 0755)
}

func WithLock(fn func() error) error {
	if err := ensureDataDir(); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	lockFile := lockPath()
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

func LoadRegistry() (*InternalServerInfo, error) {
	data, err := os.ReadFile(registryPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	var info InternalServerInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return &info, nil
}

func SaveRegistry(info *InternalServerInfo) error {
	if err := ensureDataDir(); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(registryPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

func ClearRegistry() error {
	if err := os.Remove(registryPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove registry: %w", err)
	}
	return nil
}

func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

func IsPortReachable(port int) bool {
	if port <= 0 {
		return false
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/session", port)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}
