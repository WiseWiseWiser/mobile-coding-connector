package daemon

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"time"
)

// ExitReasonType represents why the health check loop exited
type ExitReasonType string

const (
	ExitReasonProcessExit   ExitReasonType = "process exited"
	ExitReasonPortDead      ExitReasonType = "port unreachable"
	ExitReasonUpgrade       ExitReasonType = "binary upgrade"
	ExitReasonRestart       ExitReasonType = "restart requested"
	ExitReasonDaemonRestart ExitReasonType = "daemon restart requested"
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
func (hc *HealthChecker) Run(port int, cmd *exec.Cmd, currentBinPath string, findNewerBinary func(string) string) (exitReason ExitReasonType) {
	runStart := time.Now()
	pid := 0
	if cmd != nil && cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	exitReason = ExitReasonProcessExit
	defer func() {
		if recovered := recover(); recovered != nil {
			LogPanic(fmt.Sprintf("health checker (port=%d, pid=%d)", port, pid), recovered)
			exitReason = ExitReasonProcessExit
		}
		Logger("[health-check] Run exiting for port %d (PID=%d) after %v with reason: %s",
			port, pid, time.Since(runStart), exitReason)
	}()
	Logger("[health-check] Run started for port %d (PID=%d, currentBin=%s)", port, pid, currentBinPath)

	// Channel to receive process exit notification
	done := make(chan struct{}, 1)
	go func() {
		defer func() {
			LogPanic(fmt.Sprintf("health checker wait goroutine (port=%d, pid=%d)", port, pid), recover())
		}()
		Logger("[health-check] Wait goroutine started for port %d (PID=%d)", port, pid)
		cmd.Process.Wait()
		Logger("[health-check] Wait goroutine observed process exit for port %d (PID=%d)", port, pid)
		close(done)
	}()

	healthTicker := time.NewTicker(HealthCheckInterval)
	defer healthTicker.Stop()

	upgradeTicker := time.NewTicker(UpgradeCheckInterval)
	defer upgradeTicker.Stop()

	// Initialize next check time
	hc.state.SetNextHealthCheckTime(time.Now().Add(HealthCheckInterval))
	Logger("[health-check] Initial next health check scheduled at %s",
		hc.state.GetNextHealthCheckTime().Format(time.RFC3339))

	consecutiveFailures := 0

	for {
		nextScheduled := hc.state.GetNextHealthCheckTime()
		Logger("[health-check] Waiting for next event (port=%d, PID=%d, nextHealth=%s, consecutiveFailures=%d)",
			port,
			pid,
			formatMaybeTime(nextScheduled),
			consecutiveFailures)
		select {
		case <-done:
			// Process exited on its own
			Logger("Process exited (PID=%d)", pid)
			exitReason = ExitReasonProcessExit
			return exitReason

		case <-hc.state.GetRestartChannel():
			// Restart requested via API
			Logger("Restart requested via API, stopping server (PID=%d)...", pid)
			hc.gracefulStop(cmd)
			WaitForDone(done, 5*time.Second)
			exitReason = ExitReasonRestart
			return exitReason

		case <-hc.state.GetDaemonRestartChannel():
			// Daemon restart requested via API
			Logger("Daemon restart requested, stopping server (PID=%d)...", pid)
			hc.gracefulStop(cmd)
			WaitForDone(done, 5*time.Second)
			exitReason = ExitReasonDaemonRestart
			return exitReason

		case tickAt := <-healthTicker.C:
			if hc.state.IsDaemonShutdownRequested() {
				Logger("[health-check] Daemon shutdown requested, skipping health tick at %s", tickAt.Format(time.RFC3339))
				continue
			}
			Logger("[health-check] Health ticker fired at %s (port=%d, PID=%d, last scheduled=%s)",
				tickAt.Format(time.RFC3339),
				port,
				pid,
				formatMaybeTime(nextScheduled))
			// Check if health checks are paused (e.g., after exec-restart)
			if hc.state.IsHealthChecksPaused() {
				Logger("[health-check] Health checks paused, skipping this check")
				// Update next check time
				hc.state.SetNextHealthCheckTime(time.Now().Add(HealthCheckInterval))
				continue
			}

			checkStart := time.Now()
			Logger("[health-check] Starting periodic health check for port %d (PID=%d)", port, pid)

			// Update next check time
			Logger("[health-check] Updating next health check time for port %d", port)
			hc.state.SetNextHealthCheckTime(time.Now().Add(HealthCheckInterval))
			Logger("[health-check] Next health check scheduled at %s", hc.state.GetNextHealthCheckTime().Format("2006-01-02T15:04:05"))

			Logger("[health-check] Step 1/2: Checking TCP connectivity to port %d...", port)
			tcpStart := time.Now()
			if !IsPortReachable(port) {
				Logger("[health-check] Step 1/2 FAILED: TCP check failed after %v", time.Since(tcpStart))
				consecutiveFailures++
				Logger("[health-check] Consecutive failures: %d/%d", consecutiveFailures, MaxConsecutiveFailures)

				if consecutiveFailures >= MaxConsecutiveFailures {
					Logger("[health-check] CRITICAL: Port %d is not accessible after %d consecutive checks, killing server (PID=%d)...",
						port, consecutiveFailures, pid)
					hc.killProcess(cmd)
					WaitForDone(done, 5*time.Second)
					Logger("[health-check] Health check loop exiting with reason: %s", ExitReasonPortDead)
					exitReason = ExitReasonPortDead
					return exitReason
				}
				Logger("[health-check] Health check cycle completed in %v (FAILURE)", time.Since(checkStart))
			} else {
				Logger("[health-check] Step 1/2 PASSED: TCP connectivity confirmed in %v", time.Since(tcpStart))
				Logger("[health-check] Step 2/2: Checking HTTP /ping endpoint...")
				pingStart := time.Now()
				pingOK := hc.checkPingEndpoint(port)
				Logger("[health-check] Step 2/2 result: ping check completed in %v (healthy=%v)", time.Since(pingStart), pingOK)

				if consecutiveFailures > 0 {
					Logger("[health-check] RECOVERY: Port %d health check recovered after previous failures", port)
				}
				consecutiveFailures = 0
				Logger("[health-check] Health check cycle completed successfully in %v (PASSED)", time.Since(checkStart))
			}

		case tickAt := <-upgradeTicker.C:
			if hc.state.IsDaemonShutdownRequested() {
				Logger("[health-check] Daemon shutdown requested, skipping upgrade tick at %s", tickAt.Format(time.RFC3339))
				continue
			}
			Logger("[health-check] Upgrade ticker fired at %s (currentBin=%s)", tickAt.Format(time.RFC3339), currentBinPath)
			if newerBin := findNewerBinary(currentBinPath); newerBin != "" {
				Logger("Detected newer binary: %s, triggering exec restart...", newerBin)
				hc.state.SetBinPath(newerBin)
				// Call the server's exec-restart endpoint which will:
				// 1. Perform graceful shutdown
				// 2. Use syscall.Exec to replace with new binary (preserving PID)
				if callExecRestartEndpoint() {
					// Wait a bit for the exec to complete
					Logger("Waiting for server to exec with new binary...")
					WaitForDone(done, 35*time.Second)

					// Check if exec was successful by verifying:
					// 1. Server process is still alive
					// 2. /ping endpoint returns "pong"
					if hc.checkProcessAlive(port) {
						Logger("Exec-restart succeeded: server is still running with new binary")
						exitReason = ExitReasonUpgrade
						return exitReason
					}
					// Fallback: exec didn't work, need to start fresh
					Logger("Exec-restart verification failed, falling back to kill+start...")
				} else {
					Logger("Exec-restart request failed, falling back to graceful stop and restart...")
				}
				// Fallback to old behavior
				Logger("Falling back: killing old process and starting new one...")
				hc.gracefulStop(cmd)
				WaitForDone(done, 5*time.Second)
				exitReason = ExitReasonUpgrade
				return exitReason
			}
		}
	}
}

func formatMaybeTime(t time.Time) string {
	if t.IsZero() {
		return "<zero>"
	}
	return t.Format(time.RFC3339)
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

// checkPingEndpoint checks if the server's /ping endpoint returns "pong"
func (hc *HealthChecker) checkPingEndpoint(port int) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("http://localhost:%d/ping", port)
	Logger("[health-check] Making HTTP GET request to %s", url)

	resp, err := client.Get(url)
	if err != nil {
		Logger("[health-check] Ping request failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	Logger("[health-check] Ping response status: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		Logger("[health-check] Failed to read ping response body: %v", err)
		return false
	}

	bodyStr := string(body)
	Logger("[health-check] Ping response body: %q", bodyStr)
	return bodyStr == "pong"
}

// checkProcessAlive verifies that the server process is still running and responsive.
// Returns true if the server is alive and responding, false otherwise.
// This is used to verify if exec-restart was successful.
func (hc *HealthChecker) checkProcessAlive(port int) bool {
	// First check TCP connectivity
	if !IsPortReachable(port) {
		Logger("[exec-check] Port %d is not reachable", port)
		return false
	}

	// Then check /ping endpoint
	if !hc.checkPingEndpoint(port) {
		Logger("[exec-check] /ping endpoint not responding")
		return false
	}

	Logger("[exec-check] Server is alive and responding")
	return true
}

// checkTCPConnectivity checks if a port is reachable via TCP
func (hc *HealthChecker) checkTCPConnectivity(port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	Logger("[health-check] Attempting TCP connection to %s", address)

	conn, err := net.DialTimeout("tcp", address, PortCheckTimeout)
	if err != nil {
		Logger("[health-check] TCP connection failed: %v", err)
		return false
	}
	defer conn.Close()

	Logger("[health-check] TCP connection established successfully")
	return true
}
