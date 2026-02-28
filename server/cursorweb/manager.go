package cursorweb

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
)

// ServerManager manages the cursor-web process.
type ServerManager struct {
	mu      sync.RWMutex
	cmd     *exec.Cmd
	port    int
	started bool
}

// NewServerManager creates a new server manager.
func NewServerManager(port int) *ServerManager {
	if port == 0 {
		port = 3001
	}
	return &ServerManager{
		port: port,
	}
}

func processAlive(proc *os.Process) bool {
	if proc == nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// IsRunning checks if the server is running.
func (sm *ServerManager) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.started || sm.cmd == nil {
		return false
	}
	return processAlive(sm.cmd.Process)
}

// CheckServerHTTP tries to connect to the server via HTTP.
func (sm *ServerManager) CheckServerHTTP() bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(fmt.Sprintf("http://localhost:%d", sm.port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return true
}

// Start starts the Cursor Web server.
func (sm *ServerManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.started && sm.cmd != nil && processAlive(sm.cmd.Process) {
		return nil
	}

	args := []string{"@siteboon/claude-code-ui", "--port", fmt.Sprintf("%d", sm.port)}
	cmdWrapper, err := tool_exec.New("npx", args, nil)
	if err != nil {
		return fmt.Errorf("failed to resolve npx: %w", err)
	}

	cmd := cmdWrapper.Cmd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start @siteboon/claude-code-ui: %w", err)
	}

	sm.cmd = cmd
	sm.started = true
	return nil
}

// Stop stops the Cursor Web server.
func (sm *ServerManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.started {
		return nil
	}

	if sm.cmd != nil && sm.cmd.Process != nil {
		_ = sm.cmd.Process.Signal(os.Interrupt)
		time.Sleep(1 * time.Second)
		if processAlive(sm.cmd.Process) {
			_ = sm.cmd.Process.Kill()
		}
	}

	sm.started = false
	sm.cmd = nil
	return nil
}

// GetPort returns the configured port.
func (sm *ServerManager) GetPort() int {
	return sm.port
}

var globalManager *ServerManager
var globalMu sync.Once

// GetGlobalManager returns the global server manager singleton.
func GetGlobalManager() *ServerManager {
	globalMu.Do(func() {
		globalManager = NewServerManager(3001)
	})
	return globalManager
}
