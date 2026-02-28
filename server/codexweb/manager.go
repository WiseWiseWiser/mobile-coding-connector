package codexweb

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

// ServerManager manages the codex-web-local process
type ServerManager struct {
	mu      sync.RWMutex
	cmd     *exec.Cmd
	port    int
	started bool
	cancel  context.CancelFunc
}

// NewServerManager creates a new server manager
func NewServerManager(port int) *ServerManager {
	if port == 0 {
		port = 3000
	}
	return &ServerManager{
		port: port,
	}
}

// IsRunning checks if the server is running
func (sm *ServerManager) IsRunning() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.started || sm.cmd == nil {
		return false
	}

	// Check if process is still running
	if sm.cmd.Process != nil {
		// Send signal 0 to check if process exists (doesn't actually send a signal)
		err := sm.cmd.Process.Signal(os.Signal(nil))
		return err == nil
	}

	return false
}

// CheckServerHTTP tries to connect to the server via HTTP
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

// Start starts the codex-web-local server
func (sm *ServerManager) Start() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.started && sm.IsRunning() {
		return nil // Already running
	}

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	sm.cancel = cancel

	// Start codex-web-local
	// Try npx first, then look for global installation
	cmd := exec.CommandContext(ctx, "npx", "codex-web-local", "--port", fmt.Sprintf("%d", sm.port))

	// Set up process attributes
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start codex-web-local: %w", err)
	}

	sm.cmd = cmd
	sm.started = true

	// Wait a bit for the server to start
	go func() {
		time.Sleep(3 * time.Second)
	}()

	return nil
}

// Stop stops the codex-web-local server
func (sm *ServerManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.started {
		return nil
	}

	if sm.cancel != nil {
		sm.cancel()
	}

	if sm.cmd != nil && sm.cmd.Process != nil {
		// Try graceful shutdown first
		sm.cmd.Process.Signal(os.Interrupt)
		time.Sleep(1 * time.Second)

		// Force kill if still running
		if sm.cmd.Process != nil {
			sm.cmd.Process.Kill()
		}
	}

	sm.started = false
	sm.cmd = nil

	return nil
}

// GetPort returns the port number
func (sm *ServerManager) GetPort() int {
	return sm.port
}

// Global server manager instance
var globalManager *ServerManager
var globalMu sync.Once

// GetGlobalManager returns the global server manager singleton
func GetGlobalManager() *ServerManager {
	globalMu.Do(func() {
		globalManager = NewServerManager(3000)
	})
	return globalManager
}
