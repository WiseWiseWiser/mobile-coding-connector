package daemon

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

// ProcessManager handles the lifecycle of the managed server process
type ProcessManager struct {
	state   *State
	logFile *os.File
}

// NewProcessManager creates a new process manager
func NewProcessManager(state *State) *ProcessManager {
	return &ProcessManager{
		state: state,
	}
}

// OpenLogFile opens the server log file for writing
func (pm *ProcessManager) OpenLogFile() error {
	logFile, err := os.OpenFile(config.ServerLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		Logger("Warning: could not open log file: %v", err)
		return err
	}
	pm.logFile = logFile
	return nil
}

// CloseLogFile closes the log file
func (pm *ProcessManager) CloseLogFile() {
	if pm.logFile != nil {
		pm.logFile.Close()
		pm.logFile = nil
	}
}

// StartServer starts the server process and returns the command
func (pm *ProcessManager) StartServer(binPath string, serverArgs []string) (*exec.Cmd, error) {
	// Ensure the binary is executable
	os.Chmod(binPath, 0755)

	cmd := exec.Command(binPath, serverArgs...)
	cmd.Dir, _ = os.Getwd()

	// Create a new process group so we can kill all child processes
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Tee stdout/stderr to both console and log file
	if pm.logFile != nil {
		cmd.Stdout = io.MultiWriter(os.Stdout, pm.logFile)
		cmd.Stderr = io.MultiWriter(os.Stderr, pm.logFile)
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	pid := cmd.Process.Pid
	Logger("Server started (PID=%d)", pid)

	pm.state.SetServerPID(pid)
	pm.state.SetStartedAt(time.Now())

	return cmd, nil
}

// WaitForPort waits for the port to become accessible within the timeout
func (pm *ProcessManager) WaitForPort(port int, timeout time.Duration, cmd *exec.Cmd) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if IsPortReachable(port) {
			return true
		}
		// Check if process already exited
		if cmd.ProcessState != nil {
			return false
		}
		time.Sleep(1 * time.Second)
	}
	return false
}

// GracefulStopGroup sends SIGTERM to the process group first, waits, then SIGKILL
func (pm *ProcessManager) GracefulStopGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fallback: kill just the process
		Logger("Warning: could not get process group, falling back to process kill")
		cmd.Process.Signal(syscall.SIGTERM)
		time.Sleep(3 * time.Second)
		cmd.Process.Signal(syscall.SIGKILL)
		return
	}

	// Send SIGTERM to the entire process group
	Logger("Sending SIGTERM to process group %d", pgid)
	syscall.Kill(-pgid, syscall.SIGTERM)

	// Wait up to 5 seconds for graceful shutdown
	time.Sleep(5 * time.Second)

	// Force kill the entire process group
	Logger("Sending SIGKILL to process group %d", pgid)
	syscall.Kill(-pgid, syscall.SIGKILL)
}

// KillProcessGroup kills the entire process group immediately
func (pm *ProcessManager) KillProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fallback: kill just the process
		Logger("Warning: could not get process group, falling back to process kill")
		cmd.Process.Signal(syscall.SIGKILL)
		return
	}

	Logger("Killing process group %d", pgid)
	syscall.Kill(-pgid, syscall.SIGKILL)
}

// WaitForDone waits for the done signal with a timeout
func WaitForDone(done <-chan struct{}, timeout time.Duration) {
	select {
	case <-done:
		// Process exited
	case <-time.After(timeout):
		// Timeout reached
	}
}

// IsPortReachable checks if a port is reachable and the /ping endpoint returns "pong"
func IsPortReachable(port int) bool {
	// First check if port is accessible via TCP
	Logger("[IsPortReachable] Step 1/2: Checking TCP connectivity to localhost:%d (timeout=%v)", port, PortCheckTimeout)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), PortCheckTimeout)
	if err != nil {
		Logger("[IsPortReachable] TCP connection failed: %v", err)
		return false
	}
	conn.Close()
	Logger("[IsPortReachable] TCP connection successful")

	// Then verify /ping endpoint returns "pong" within 5 seconds
	Logger("[IsPortReachable] Step 2/2: Checking HTTP /ping endpoint")
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://localhost:%d/ping", port)
	Logger("[IsPortReachable] Making HTTP GET request to %s", url)

	resp, err := client.Get(url)
	if err != nil {
		Logger("[IsPortReachable] HTTP ping request failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	Logger("[IsPortReachable] HTTP response status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		Logger("[IsPortReachable] HTTP response status is not 200 OK")
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		Logger("[IsPortReachable] Failed to read HTTP response body: %v", err)
		return false
	}

	bodyStr := string(body)
	isPong := bodyStr == "pong"
	Logger("[IsPortReachable] HTTP response body: %q (isPong=%v)", bodyStr, isPong)
	return isPong
}

// FindPortPID finds the PID using a specific port (for conflict detection)
func FindPortPID(port int) string {
	// Try lsof first
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port))
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return string(output)
	}

	// Try netstat as fallback
	cmd = exec.Command("sh", "-c", fmt.Sprintf("netstat -tlnp 2>/dev/null | grep ':%d ' | awk '{print $7}' | cut -d'/' -f1", port))
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		return string(output)
	}

	return ""
}

// IsPortInUse checks if a port is already in use
func IsPortInUse(port int) bool {
	return IsPortReachable(port)
}
