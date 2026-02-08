package run

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
	"github.com/xhd2015/lifelog-private/ai-critic/server/terminal"
)

const (
	startupTimeout        = 10 * time.Second
	healthCheckInterval   = 10 * time.Second
	restartDelay          = 3 * time.Second
	portCheckTimeout      = 2 * time.Second
	upgradeCheckInterval  = 30 * time.Second
)

func runKeepAlive(args []string) error {
	var scriptFlag bool
	var portFlag int
	var foreverFlag bool
	args, err := flags.
		Bool("--script", &scriptFlag).
		Int("--port", &portFlag).
		Bool("--forever", &foreverFlag).
		Parse(args)
	if err != nil {
		return err
	}

	port := lib.DefaultServerPort
	if portFlag > 0 {
		port = portFlag
	}

	if scriptFlag {
		return outputKeepAliveScript(port, args)
	}
	return runKeepAliveLoop(port, foreverFlag, args)
}

// runKeepAliveLoop implements the keep-alive logic in Go.
func runKeepAliveLoop(port int, forever bool, serverArgs []string) error {
	// Check if port is already in use - another keep-alive is likely running
	// Skip this check if --forever flag is set
	if !forever && isPortReachable(port) {
		pid := findPortPID(port)
		if pid != "" {
			return fmt.Errorf("port %d is already in use by PID %s - another keep-alive instance may be running", port, pid)
		}
		return fmt.Errorf("port %d is already in use - another keep-alive instance may be running", port)
	}

	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	logFile, err := os.OpenFile("ai-critic-server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("[%s] Warning: could not open log file: %v\n", timestamp(), err)
		logFile = nil
	}
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	for {
		// Before starting, check if there's a newer versioned binary
		newerBin := findNewerBinary(binPath)
		if newerBin != "" {
			fmt.Printf("[%s] Found newer binary: %s (upgrading from %s)\n", timestamp(), newerBin, filepath.Base(binPath))
			binPath = newerBin
		}

		fmt.Printf("[%s] Starting ai-critic server on port %d (binary: %s)...\n", timestamp(), port, filepath.Base(binPath))

		// Build server args: include --port if it was specified
		cmdArgs := append([]string{}, serverArgs...)

		// Ensure the binary is executable
		os.Chmod(binPath, 0755)

		cmd := exec.Command(binPath, cmdArgs...)
		cmd.Dir, _ = os.Getwd()

		// Tee stdout/stderr to both console and log file
		if logFile != nil {
			cmd.Stdout = io.MultiWriter(os.Stdout, logFile)
			cmd.Stderr = io.MultiWriter(os.Stderr, logFile)
		} else {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		if err := cmd.Start(); err != nil {
			fmt.Printf("[%s] Failed to start server: %v\n", timestamp(), err)
			fmt.Printf("[%s] Restarting in %v...\n", timestamp(), restartDelay)
			time.Sleep(restartDelay)
			continue
		}

		pid := cmd.Process.Pid
		fmt.Printf("[%s] Server started (PID=%d)\n", timestamp(), pid)

		// Wait for port to become ready
		ready := waitForPort(port, startupTimeout, cmd)
		if !ready {
			fmt.Printf("[%s] ERROR: Server failed to become ready within %v\n", timestamp(), startupTimeout)
			killProcess(cmd)
			fmt.Printf("[%s] Restarting in %v...\n", timestamp(), restartDelay)
			time.Sleep(restartDelay)
			continue
		}

		fmt.Printf("[%s] Server is ready (PID=%d, port=%d)\n", timestamp(), pid, port)

		// Health check loop (also checks for binary upgrades)
		exitReason := healthCheckLoop(port, cmd, binPath)

		switch exitReason {
		case exitReasonUpgrade:
			fmt.Printf("[%s] Upgrading binary, restarting immediately...\n", timestamp())
		default:
			fmt.Printf("[%s] Server exited (%s), restarting in %v...\n", timestamp(), exitReason, restartDelay)
			time.Sleep(restartDelay)
		}
	}
}

type exitReasonType string

const (
	exitReasonProcessExit  exitReasonType = "process exited"
	exitReasonPortDead     exitReasonType = "port unreachable"
	exitReasonUpgrade      exitReasonType = "binary upgrade"
)

// waitForPort waits for the port to become accessible within the timeout.
// Returns false if the process exits or the timeout is reached.
func waitForPort(port int, timeout time.Duration, cmd *exec.Cmd) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isPortReachable(port) {
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

// healthCheckLoop periodically checks port accessibility and for binary upgrades.
// Returns the reason the loop ended.
func healthCheckLoop(port int, cmd *exec.Cmd, currentBinPath string) exitReasonType {
	// Channel to receive process exit
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	healthTicker := time.NewTicker(healthCheckInterval)
	defer healthTicker.Stop()

	upgradeTicker := time.NewTicker(upgradeCheckInterval)
	defer upgradeTicker.Stop()

	consecutiveFailures := 0
	const maxConsecutiveFailures = 2

	for {
		select {
		case <-done:
			// Process exited on its own
			return exitReasonProcessExit

		case <-healthTicker.C:
			if !isPortReachable(port) {
				consecutiveFailures++
				fmt.Printf("[%s] Port %d health check failed (%d/%d)\n", timestamp(), port, consecutiveFailures, maxConsecutiveFailures)

				if consecutiveFailures >= maxConsecutiveFailures {
					fmt.Printf("[%s] Port %d is not accessible after %d checks, killing server (PID=%d)...\n",
						timestamp(), port, consecutiveFailures, cmd.Process.Pid)
					killProcess(cmd)
					<-done // Wait for process goroutine to finish
					return exitReasonPortDead
				}
			} else {
				consecutiveFailures = 0
			}

		case <-upgradeTicker.C:
			newerBin := findNewerBinary(currentBinPath)
			if newerBin != "" {
				fmt.Printf("[%s] Detected newer binary: %s, stopping current server for upgrade...\n",
					timestamp(), filepath.Base(newerBin))
				gracefulStop(cmd)
				<-done // Wait for process goroutine to finish
				return exitReasonUpgrade
			}
		}
	}
}

// gracefulStop sends SIGTERM first, waits briefly, then SIGKILL.
func gracefulStop(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	// Try graceful shutdown first
	cmd.Process.Signal(syscall.SIGTERM)

	// Wait up to 5 seconds for graceful shutdown
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	exitCh := make(chan struct{})
	go func() {
		cmd.Process.Wait()
		close(exitCh)
	}()

	select {
	case <-exitCh:
		return
	case <-timer.C:
		// Force kill
		cmd.Process.Signal(syscall.SIGKILL)
		<-exitCh
	}
}

func isPortReachable(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), portCheckTimeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Signal(syscall.SIGKILL)
	}
}

func timestamp() string {
	return time.Now().Format("2006-01-02T15:04:05")
}

// ---- Binary Version Upgrade ----

// versionRegex matches -vN at the end of a binary name (before any extension).
// e.g. "ai-critic-server-linux-amd64-v4" -> version 4
var versionRegex = regexp.MustCompile(`-v(\d+)$`)

// parseBinVersion extracts the base name and version from a binary path.
// Returns (baseName, version). If no -vN suffix, version is 0.
// e.g. "ai-critic-server-linux-amd64"     -> ("ai-critic-server-linux-amd64", 0)
// e.g. "ai-critic-server-linux-amd64-v4"  -> ("ai-critic-server-linux-amd64", 4)
func parseBinVersion(binPath string) (baseName string, version int) {
	name := filepath.Base(binPath)

	match := versionRegex.FindStringSubmatch(name)
	if match == nil {
		return name, 0
	}

	v, err := strconv.Atoi(match[1])
	if err != nil {
		return name, 0
	}

	baseName = name[:len(name)-len(match[0])]
	return baseName, v
}

// findNewerBinary looks for a newer versioned binary in the same directory.
// Returns the full path to the newer binary, or empty string if none found.
func findNewerBinary(currentBinPath string) string {
	dir := filepath.Dir(currentBinPath)
	currentBase, currentVersion := parseBinVersion(currentBinPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	type candidate struct {
		path    string
		version int
	}

	var candidates []candidate
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Must start with the same base name
		if !strings.HasPrefix(name, currentBase) {
			continue
		}

		// Parse version
		entryBase, entryVersion := parseBinVersion(filepath.Join(dir, name))
		if entryBase != currentBase {
			continue
		}

		// Must be strictly newer
		if entryVersion <= currentVersion {
			continue
		}

		// Must be executable (non-zero size)
		info, err := entry.Info()
		if err != nil || info.Size() == 0 {
			continue
		}

		candidates = append(candidates, candidate{
			path:    filepath.Join(dir, name),
			version: entryVersion,
		})
	}

	if len(candidates) == 0 {
		return ""
	}

	// Sort by version descending, pick the highest
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].version > candidates[j].version
	})

	return candidates[0].path
}

// outputKeepAliveScript outputs a shell script for keep-alive.
func outputKeepAliveScript(port int, serverArgs []string) error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	// Build the server command with all original args
	var cmdParts []string
	cmdParts = append(cmdParts, terminal.ShellQuote(binPath))
	if port != lib.DefaultServerPort {
		cmdParts = append(cmdParts, "--port", fmt.Sprintf("%d", port))
	}
	for _, a := range serverArgs {
		cmdParts = append(cmdParts, terminal.ShellQuote(a))
	}
	serverCmd := strings.Join(cmdParts, " ")

	script := fmt.Sprintf(`#!/bin/sh
LOG_FILE="ai-critic-server.log"
PORT=%d
STARTUP_TIMEOUT=10
HEALTH_CHECK_INTERVAL=10
RESTART_DELAY=3

check_port() {
  # Returns 0 if port is reachable
  if command -v nc >/dev/null 2>&1; then
    nc -z localhost "$PORT" 2>/dev/null
  elif command -v curl >/dev/null 2>&1; then
    curl -sf --max-time 2 "http://localhost:$PORT/api/ping" >/dev/null 2>&1
  else
    # Fallback: use the binary's built-in check-port command
    %s check-port --port "$PORT" --timeout 2 2>/dev/null
  fi
}

while true; do
  echo "[$(date)] Starting ai-critic server on port $PORT..."

  # Start server in background with optional logging
  if command -v tee >/dev/null 2>&1; then
    %s 2>&1 | tee -a "$LOG_FILE" &
  else
    %s 2>&1 &
  fi
  SERVER_PID=$!

  # Wait for port to become ready (max STARTUP_TIMEOUT seconds)
  WAITED=0
  READY=0
  while [ "$WAITED" -lt "$STARTUP_TIMEOUT" ]; do
    sleep 1
    WAITED=$((WAITED + 1))
    if check_port; then
      READY=1
      break
    fi
    if ! kill -0 "$SERVER_PID" 2>/dev/null; then
      echo "[$(date)] Server process $SERVER_PID exited during startup"
      break
    fi
  done

  if [ "$READY" -ne 1 ]; then
    echo "[$(date)] ERROR: Server failed to become ready within ${STARTUP_TIMEOUT}s"
    kill -9 "$SERVER_PID" 2>/dev/null
    wait "$SERVER_PID" 2>/dev/null
    echo "[$(date)] Restarting in ${RESTART_DELAY}s..."
    sleep "$RESTART_DELAY"
    continue
  fi

  echo "[$(date)] Server is ready (PID=$SERVER_PID, port=$PORT)"

  while true; do
    sleep "$HEALTH_CHECK_INTERVAL"
    if ! kill -0 "$SERVER_PID" 2>/dev/null; then
      echo "[$(date)] Server process $SERVER_PID is no longer running"
      break
    fi
    if ! check_port; then
      echo "[$(date)] Port $PORT is not accessible, killing server (PID=$SERVER_PID)..."
      kill -9 "$SERVER_PID" 2>/dev/null
      wait "$SERVER_PID" 2>/dev/null
      break
    fi
  done

  wait "$SERVER_PID" 2>/dev/null
  EXIT_CODE=$?
  echo "[$(date)] Server exited with code $EXIT_CODE, restarting in ${RESTART_DELAY}s..."
  sleep "$RESTART_DELAY"
done
`, port, terminal.ShellQuote(binPath), serverCmd, serverCmd)

	fmt.Print(script)
	return nil
}
