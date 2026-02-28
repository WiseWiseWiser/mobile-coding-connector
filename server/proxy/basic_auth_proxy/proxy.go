package basic_auth_proxy

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/proc_manager"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

const (
	procName   = "basic-auth-proxy"
	binaryName = "basic-auth-proxy"
)

type proxyConfig struct {
	BackendPort int `json:"backend_port"`
}

func configPath() string {
	return filepath.Join(config.DataDir, "basic-auth-proxy.json")
}

func IsRunning(port int) bool {
	if port <= 0 {
		return false
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func GetBackendPort() int {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return 0
	}

	var cfg proxyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0
	}
	return cfg.BackendPort
}

func SaveBackendPort(backendPort int) error {
	if backendPort <= 0 {
		return fmt.Errorf("backend port must be > 0, got: %d", backendPort)
	}

	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	data, err := json.MarshalIndent(proxyConfig{BackendPort: backendPort}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal proxy config: %w", err)
	}
	if err := os.WriteFile(configPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write proxy config: %w", err)
	}
	return nil
}

func RemoveConfig() error {
	err := os.Remove(configPath())
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove proxy config: %w", err)
	}
	return nil
}

func Start(proxyPort, backendPort int) error {
	if proxyPort <= 0 {
		return fmt.Errorf("proxy port must be > 0, got: %d", proxyPort)
	}
	if backendPort <= 0 {
		return fmt.Errorf("backend port must be > 0, got: %d", backendPort)
	}

	return proc_manager.WithLock(procName, func() error {
		reg, err := proc_manager.LoadRegistry(procName)
		if err != nil {
			fmt.Printf("[basic_auth_proxy] Warning: failed to load registry: %v\n", err)
		}

		if reg != nil && reg.PID > 0 && proc_manager.IsProcessAlive(reg.PID) {
			if reg.Port == proxyPort && proc_manager.IsPortReachable(reg.Port, "") {
				if err := SaveBackendPort(backendPort); err != nil {
					return err
				}
				fmt.Printf("[basic_auth_proxy] Reusing existing proxy: PID=%d, Port=%d\n", reg.PID, reg.Port)
				return nil
			}

			fmt.Printf("[basic_auth_proxy] Existing proxy not usable (PID=%d, Port=%d), stopping\n", reg.PID, reg.Port)
			_ = proc_manager.StopProcess(reg.PID)
			_ = proc_manager.ClearRegistry(procName)
		}

		if !tool_resolve.IsAvailable(binaryName) {
			return fmt.Errorf("%s binary not found in PATH", binaryName)
		}

		cmd, err := tool_exec.New(binaryName, []string{
			"--port", fmt.Sprintf("%d", proxyPort),
			"--backend-port", fmt.Sprintf("%d", backendPort),
		}, nil)
		if err != nil {
			return fmt.Errorf("failed to create proxy command: %w", err)
		}

		if err := cmd.Cmd.Start(); err != nil {
			return fmt.Errorf("failed to start proxy: %w", err)
		}

		if err := proc_manager.SaveRegistry(procName, &proc_manager.ProcessRegistry{
			Name:      procName,
			PID:       cmd.Cmd.Process.Pid,
			Port:      proxyPort,
			StartTime: time.Now().Unix(),
		}); err != nil {
			fmt.Printf("[basic_auth_proxy] Warning: failed to save registry: %v\n", err)
		}

		if err := SaveBackendPort(backendPort); err != nil {
			_ = proc_manager.StopProcess(cmd.Cmd.Process.Pid)
			_ = proc_manager.ClearRegistry(procName)
			return err
		}

		fmt.Printf("[basic_auth_proxy] Proxy started on port %d (backend: %d), PID: %d\n", proxyPort, backendPort, cmd.Cmd.Process.Pid)
		return nil
	})
}

func Stop() error {
	return proc_manager.WithLock(procName, func() error {
		reg, err := proc_manager.LoadRegistry(procName)
		if err != nil {
			fmt.Printf("[basic_auth_proxy] Warning: failed to load registry: %v\n", err)
		}

		if reg != nil && reg.PID > 0 && proc_manager.IsProcessAlive(reg.PID) {
			fmt.Printf("[basic_auth_proxy] Stopping proxy: PID=%d, Port=%d\n", reg.PID, reg.Port)
			if err := proc_manager.StopProcess(reg.PID); err != nil {
				fmt.Printf("[basic_auth_proxy] Warning: failed to stop process %d: %v\n", reg.PID, err)
			}
		}

		if err := proc_manager.ClearRegistry(procName); err != nil {
			return err
		}
		return nil
	})
}
