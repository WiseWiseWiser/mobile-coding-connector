// Package daemon implements the keep-alive daemon for managing the ai-critic server process.
// It provides automatic restarts, health checks, binary upgrades, and a management HTTP API.
package daemon

import (
	"fmt"
	"sync"
	"time"
)

// Constants for timing configuration
const (
	StartupTimeout         = 10 * time.Second
	HealthCheckInterval    = 10 * time.Second
	RestartDelay           = 3 * time.Second
	PortCheckTimeout       = 2 * time.Second
	UpgradeCheckInterval   = 30 * time.Second
	MaxConsecutiveFailures = 2
)

// State represents the daemon's mutable state with thread-safe access
type State struct {
	mu                  sync.RWMutex
	binPath             string    // current binary being run
	daemonBinPath       string    // the daemon's own binary path
	serverPort          int       // the port the managed server listens on
	serverPID           int       // PID of the currently running server, 0 if not running
	startedAt           time.Time // when the current server was started
	nextHealthCheckTime time.Time
	restartCount        int // how many times the server has been restarted
	restartCh           chan struct{}
}

// NewState creates a new daemon state instance
func NewState() *State {
	return &State{
		restartCh: make(chan struct{}, 1),
	}
}

// GetBinPath returns the current binary path (thread-safe)
func (s *State) GetBinPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.binPath
}

// SetBinPath sets the current binary path (thread-safe)
func (s *State) SetBinPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.binPath = path
}

// GetDaemonBinPath returns the daemon's own binary path (thread-safe)
func (s *State) GetDaemonBinPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.daemonBinPath
}

// SetDaemonBinPath sets the daemon's own binary path (thread-safe)
func (s *State) SetDaemonBinPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.daemonBinPath = path
}

// GetRestartCount returns the number of times the server has been restarted (thread-safe)
func (s *State) GetRestartCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.restartCount
}

// IncrementRestartCount increments the restart count (thread-safe)
func (s *State) IncrementRestartCount() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restartCount++
}

// GetServerPort returns the server port (thread-safe)
func (s *State) GetServerPort() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.serverPort
}

// SetServerPort sets the server port (thread-safe)
func (s *State) SetServerPort(port int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serverPort = port
}

// GetServerPID returns the server PID (thread-safe)
func (s *State) GetServerPID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.serverPID
}

// SetServerPID sets the server PID (thread-safe)
func (s *State) SetServerPID(pid int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.serverPID = pid
}

// GetStartedAt returns when the server was started (thread-safe)
func (s *State) GetStartedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.startedAt
}

// SetStartedAt sets when the server was started (thread-safe)
func (s *State) SetStartedAt(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.startedAt = t
}

// GetNextHealthCheckTime returns the next health check time (thread-safe)
func (s *State) GetNextHealthCheckTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.nextHealthCheckTime
}

// SetNextHealthCheckTime sets the next health check time (thread-safe)
func (s *State) SetNextHealthCheckTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextHealthCheckTime = t
}

// RequestRestart signals the daemon to restart the server (non-blocking)
func (s *State) RequestRestart() bool {
	select {
	case s.restartCh <- struct{}{}:
		return true
	default:
		return false
	}
}

// GetRestartChannel returns the restart channel for listening
func (s *State) GetRestartChannel() <-chan struct{} {
	return s.restartCh
}

// GetStatusSnapshot returns a snapshot of current state for API responses
func (s *State) GetStatusSnapshot() StatusSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return StatusSnapshot{
		BinPath:             s.binPath,
		DaemonBinPath:       s.daemonBinPath,
		ServerPort:          s.serverPort,
		ServerPID:           s.serverPID,
		StartedAt:           s.startedAt,
		NextHealthCheckTime: s.nextHealthCheckTime,
		RestartCount:        s.restartCount,
	}
}

// StatusSnapshot is a read-only snapshot of daemon state
type StatusSnapshot struct {
	BinPath             string
	DaemonBinPath       string
	ServerPort          int
	ServerPID           int
	StartedAt           time.Time
	NextHealthCheckTime time.Time
	RestartCount        int
}

// GlobalState is the singleton daemon state instance
var GlobalState = NewState()

// Logger provides timestamped logging
func Logger(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02T15:04:05")
	fmt.Printf("[%s] %s\n", timestamp, fmt.Sprintf(format, args...))
}
