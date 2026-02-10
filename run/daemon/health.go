package daemon

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

// ExitReasonType represents why the health check loop exited
type ExitReasonType string

const (
	ExitReasonProcessExit ExitReasonType = "process exited"
	ExitReasonPortDead    ExitReasonType = "port unreachable"
	ExitReasonUpgrade     ExitReasonType = "binary upgrade"
	ExitReasonRestart     ExitReasonType = "restart requested"
)

// HealthChecker handles health checking and monitoring of the server process
type HealthChecker struct {
	state *State
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(state *State) *HealthChecker {
	return &HealthChecker{
		state: state,
	}
}

// Run starts the health check loop and returns when the server should be restarted
func (hc *HealthChecker) Run(port int, cmd *exec.Cmd, currentBinPath string, findNewerBinary func(string) string) ExitReasonType {
	// Channel to receive process exit notification
	done := make(chan struct{}, 1)
	go func() {
		cmd.Process.Wait()
		close(done)
	}()

	healthTicker := time.NewTicker(HealthCheckInterval)
	defer healthTicker.Stop()

	upgradeTicker := time.NewTicker(UpgradeCheckInterval)
	defer upgradeTicker.Stop()

	// Initialize next check time
	hc.state.SetNextHealthCheckTime(time.Now().Add(HealthCheckInterval))

	consecutiveFailures := 0

	for {
		select {
		case <-done:
			// Process exited on its own
			Logger("Process exited (PID=%d)", cmd.Process.Pid)
			return ExitReasonProcessExit

		case <-hc.state.GetRestartChannel():
			// Restart requested via API
			Logger("Restart requested via API, stopping server (PID=%d)...", cmd.Process.Pid)
			hc.gracefulStop(cmd)
			WaitForDone(done, 5*time.Second)
			return ExitReasonRestart

		case <-healthTicker.C:
			// Update next check time
			hc.state.SetNextHealthCheckTime(time.Now().Add(HealthCheckInterval))

			if !IsPortReachable(port) {
				consecutiveFailures++
				Logger("Port %d health check failed (%d/%d)", port, consecutiveFailures, MaxConsecutiveFailures)

				if consecutiveFailures >= MaxConsecutiveFailures {
					Logger("Port %d is not accessible after %d checks, killing server (PID=%d)...",
						port, consecutiveFailures, cmd.Process.Pid)
					hc.killProcess(cmd)
					WaitForDone(done, 5*time.Second)
					return ExitReasonPortDead
				}
			} else {
				if consecutiveFailures > 0 {
					Logger("Port %d health check recovered", port)
				}
				consecutiveFailures = 0
			}

		case <-upgradeTicker.C:
			if newerBin := findNewerBinary(currentBinPath); newerBin != "" {
				Logger("Detected newer binary: %s, stopping current server for upgrade...", newerBin)
				hc.state.SetBinPath(newerBin)
				hc.gracefulStop(cmd)
				WaitForDone(done, 5*time.Second)
				return ExitReasonUpgrade
			}
		}
	}
}

// gracefulStop performs a graceful shutdown of the process
func (hc *HealthChecker) gracefulStop(cmd *exec.Cmd) {
	// Try to call the shutdown endpoint first
	if CallShutdownEndpoint() {
		// Wait for server to shut down gracefully
		Logger("Shutdown request sent to server, waiting for graceful shutdown...")
		WaitForDone(make(chan struct{}), 30*time.Second)
		Logger("Graceful shutdown completed")
	} else {
		// Fallback to process group kill
		Logger("Shutdown endpoint unavailable, using process group kill")
		pm := NewProcessManager(hc.state)
		pm.GracefulStopGroup(cmd)
	}
}

// killProcess kills the process immediately
func (hc *HealthChecker) killProcess(cmd *exec.Cmd) {
	pm := NewProcessManager(hc.state)
	pm.KillProcessGroup(cmd)
}

// CallShutdownEndpoint calls the server's shutdown endpoint with auth.
// Returns true if the request was successful.
func CallShutdownEndpoint() bool {
	token, err := loadFirstToken()
	if err != nil {
		Logger("Failed to load auth token: %v", err)
		return false
	}

	port := config.DefaultServerPort
	url := fmt.Sprintf("http://localhost:%d/api/shutdown", port)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		Logger("Failed to create shutdown request: %v", err)
		return false
	}

	// Add auth cookie
	if token != "" {
		req.AddCookie(&http.Cookie{
			Name:  "ai-critic-token",
			Value: token,
		})
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		Logger("Failed to call shutdown endpoint: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		Logger("Shutdown endpoint returned success")
		return true
	}

	Logger("Shutdown endpoint returned status: %d", resp.StatusCode)
	return false
}

// loadFirstToken reads the first non-empty line from the credentials file.
func loadFirstToken() (string, error) {
	f, err := os.Open(config.CredentialsFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line, nil
		}
	}
	return "", scanner.Err()
}
