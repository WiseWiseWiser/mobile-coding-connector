// Package subprocess manages asynchronous subprocesses with lifecycle control.
// It provides a centralized way to start, monitor, and stop background processes.
package subprocess

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Manager manages multiple subprocesses
var (
	globalManager *Manager
	once          sync.Once
)

// GetManager returns the global subprocess manager singleton
func GetManager() *Manager {
	once.Do(func() {
		globalManager = NewManager()
	})
	return globalManager
}

// Process represents a managed subprocess
type Process struct {
	ID        string
	Name      string
	Cmd       *exec.Cmd
	Status    ProcessStatus
	StartTime time.Time
	StopTime  *time.Time
	ExitCode  *int
	Error     error

	// Control channels
	stopChan chan struct{}
	doneChan chan struct{}

	// Health check
	HealthChecker func() bool
}

// ProcessStatus represents the status of a process
type ProcessStatus string

const (
	StatusPending  ProcessStatus = "pending"
	StatusStarting ProcessStatus = "starting"
	StatusRunning  ProcessStatus = "running"
	StatusStopping ProcessStatus = "stopping"
	StatusStopped  ProcessStatus = "stopped"
	StatusError    ProcessStatus = "error"
)

// Manager manages multiple subprocesses
type Manager struct {
	mu        sync.RWMutex
	processes map[string]*Process
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewManager creates a new subprocess manager
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		processes: make(map[string]*Process),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// StartProcess starts a new managed subprocess
// The process will run in its own process group and won't block
func (m *Manager) StartProcess(id string, name string, cmd *exec.Cmd, healthChecker func() bool, detach ...bool) (*Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if process already exists
	if existing, ok := m.processes[id]; ok {
		if existing.Status == StatusRunning {
			return existing, fmt.Errorf("process %s (id=%s) is already running", name, id)
		}
		// Clean up stopped process
		delete(m.processes, id)
	}

	// Create process group so it won't receive parent's signals
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.SysProcAttr.Pgid = 0

	process := &Process{
		ID:            id,
		Name:          name,
		Cmd:           cmd,
		Status:        StatusStarting,
		StartTime:     time.Now(),
		stopChan:      make(chan struct{}),
		doneChan:      make(chan struct{}),
		HealthChecker: healthChecker,
	}

	m.processes[id] = process

	// Start the process
	if err := cmd.Start(); err != nil {
		process.Status = StatusError
		process.Error = err
		close(process.doneChan)
		return nil, fmt.Errorf("failed to start process %s: %w", name, err)
	}

	process.Status = StatusRunning

	// Monitor the process in a goroutine
	go m.monitorProcess(process)

	return process, nil
}

// monitorProcess monitors a running process
func (m *Manager) monitorProcess(p *Process) {
	defer close(p.doneChan)

	// Wait for process to exit or stop signal
	done := make(chan error, 1)
	go func() {
		done <- p.Cmd.Wait()
	}()

	select {
	case err := <-done:
		m.mu.Lock()
		now := time.Now()
		p.StopTime = &now
		if err != nil {
			p.Status = StatusError
			p.Error = err
			if exitErr, ok := err.(*exec.ExitError); ok {
				code := exitErr.ExitCode()
				p.ExitCode = &code
			}
		} else {
			p.Status = StatusStopped
			code := 0
			p.ExitCode = &code
		}
		m.mu.Unlock()

	case <-p.stopChan:
		// Stop requested, kill the process
		m.mu.Lock()
		p.Status = StatusStopping
		m.mu.Unlock()

		// Kill the entire process group
		if p.Cmd.Process != nil {
			syscall.Kill(-p.Cmd.Process.Pid, syscall.SIGTERM)

			// Wait for graceful shutdown
			select {
			case <-done:
				// Process exited
			case <-time.After(5 * time.Second):
				// Force kill after timeout
				if p.Cmd.Process != nil {
					syscall.Kill(-p.Cmd.Process.Pid, syscall.SIGKILL)
				}
				<-done
			}
		}

		m.mu.Lock()
		now := time.Now()
		p.StopTime = &now
		p.Status = StatusStopped
		code := -1
		p.ExitCode = &code
		m.mu.Unlock()

	case <-m.ctx.Done():
		// Manager is shutting down, stop all processes
		m.mu.Lock()
		p.Status = StatusStopping
		m.mu.Unlock()

		if p.Cmd.Process != nil {
			syscall.Kill(-p.Cmd.Process.Pid, syscall.SIGTERM)
			<-done
		}

		m.mu.Lock()
		now := time.Now()
		p.StopTime = &now
		p.Status = StatusStopped
		m.mu.Unlock()
	}
}

// StopProcess stops a running process by ID
func (m *Manager) StopProcess(id string) error {
	m.mu.Lock()
	p, ok := m.processes[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("process %s not found", id)
	}

	if p.Status != StatusRunning && p.Status != StatusStarting {
		m.mu.Unlock()
		return fmt.Errorf("process %s is not running (status: %s)", id, p.Status)
	}
	m.mu.Unlock()

	// Signal the process to stop
	close(p.stopChan)

	// Wait for it to actually stop
	select {
	case <-p.doneChan:
		return nil
	case <-time.After(10 * time.Second):
		return fmt.Errorf("timeout waiting for process %s to stop", id)
	}
}

// GetProcess returns a process by ID
func (m *Manager) GetProcess(id string) *Process {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processes[id]
}

// GetProcessStatus returns the status of a process
func (m *Manager) GetProcessStatus(id string) (ProcessStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.processes[id]
	if !ok {
		return "", false
	}
	return p.Status, true
}

// IsRunning checks if a process is currently running
func (m *Manager) IsRunning(id string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.processes[id]
	if !ok {
		return false
	}
	return p.Status == StatusRunning
}

// ListProcesses returns all managed processes
func (m *Manager) ListProcesses() []*Process {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Process, 0, len(m.processes))
	for _, p := range m.processes {
		result = append(result, p)
	}
	return result
}

// StopAll stops all running processes
func (m *Manager) StopAll() {
	m.mu.RLock()
	processes := make([]*Process, 0, len(m.processes))
	for _, p := range m.processes {
		if p.Status == StatusRunning || p.Status == StatusStarting {
			processes = append(processes, p)
		}
	}
	m.mu.RUnlock()

	// Cancel context to signal all processes
	m.cancel()

	// Wait for all processes to stop
	var wg sync.WaitGroup
	for _, p := range processes {
		wg.Add(1)
		go func(proc *Process) {
			defer wg.Done()
			select {
			case <-proc.doneChan:
				return
			case <-time.After(10 * time.Second):
				// Force kill if needed
				if proc.Cmd.Process != nil {
					syscall.Kill(-proc.Cmd.Process.Pid, syscall.SIGKILL)
				}
			}
		}(p)
	}
	wg.Wait()
}

// Cleanup removes stopped processes from the manager
func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, p := range m.processes {
		if p.Status == StatusStopped || p.Status == StatusError {
			delete(m.processes, id)
		}
	}
}

// String returns a string representation of process status
func (s ProcessStatus) String() string {
	return string(s)
}

// GetUptime returns the uptime of a running process
func (p *Process) GetUptime() time.Duration {
	if p.Status != StatusRunning {
		return 0
	}
	return time.Since(p.StartTime)
}

// WaitForRunning waits for a process to be running by checking the health checker
// Returns true if health check passes, false if timeout
func (p *Process) WaitForRunning(timeout time.Duration) bool {
	if p.HealthChecker == nil {
		// No health checker, just wait a bit for process to start
		time.Sleep(500 * time.Millisecond)
		return true
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p.HealthChecker() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// LogOutput logs process output to a file
func (p *Process) LogOutput(stdout, stderr *os.File) {
	if p.Cmd == nil {
		return
	}
	p.Cmd.Stdout = stdout
	p.Cmd.Stderr = stderr
}
