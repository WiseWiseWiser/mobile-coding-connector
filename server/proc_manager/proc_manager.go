package proc_manager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

type ProcessRegistry struct {
	Name       string `json:"name"`
	PID        int    `json:"pid"`
	Port       int    `json:"port"`
	StartTime  int64  `json:"start_time"`
	CustomPath string `json:"custom_path,omitempty"`
}

func ensureDir(name string) error {
	dir := getDir(name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create proc directory %s: %w", name, err)
	}
	return nil
}

func getDir(name string) string {
	switch name {
	case "opencode-internal":
		return config.OpencodeInternalServerDir()
	case "opencode-web":
		return config.OpencodeWebServerDir()
	case "basic-auth-proxy":
		return config.BasicAuthProxyDir()
	default:
		return config.ProcsDir + "/" + name
	}
}

func getLockPath(name string) string {
	switch name {
	case "opencode-internal":
		return config.OpencodeInternalServerLockPath()
	case "opencode-web":
		return config.OpencodeWebServerLockPath()
	case "basic-auth-proxy":
		return config.BasicAuthProxyLockPath()
	default:
		return getDir(name) + "/lock"
	}
}

func getRegistryPath(name string) string {
	switch name {
	case "opencode-internal":
		return config.OpencodeInternalServerRegistryPath()
	case "opencode-web":
		return config.OpencodeWebServerRegistryPath()
	case "basic-auth-proxy":
		return config.BasicAuthProxyRegistryPath()
	default:
		return getDir(name) + "/registry.json"
	}
}

func EnsureDir(name string) error {
	return ensureDir(name)
}

func WithLock(name string, fn func() error) error {
	if err := ensureDir(name); err != nil {
		return fmt.Errorf("failed to ensure proc directory: %w", err)
	}

	lockFile := getLockPath(name)
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

func LoadRegistry(name string) (*ProcessRegistry, error) {
	path := getRegistryPath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	var reg ProcessRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("failed to parse registry: %w", err)
	}

	return &reg, nil
}

func SaveRegistry(name string, reg *ProcessRegistry) error {
	if err := ensureDir(name); err != nil {
		return err
	}

	path := getRegistryPath(name)
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

func ClearRegistry(name string) error {
	path := getRegistryPath(name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
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

func IsPortReachable(port int, path string) bool {
	if port <= 0 {
		return false
	}
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}

func StopProcess(pid int) error {
	if pid <= 0 {
		return nil
	}

	err := syscall.Kill(pid, syscall.SIGTERM)
	if err != nil {
		err = syscall.Kill(pid, syscall.SIGKILL)
		if err != nil {
			return fmt.Errorf("failed to kill process %d: %w", pid, err)
		}
	}
	return nil
}
