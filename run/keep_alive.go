package run

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
	"github.com/xhd2015/lifelog-private/ai-critic/server/terminal"
)

const (
	startupTimeout       = 10 * time.Second
	healthCheckInterval  = 10 * time.Second
	restartDelay         = 3 * time.Second
	portCheckTimeout     = 2 * time.Second
)

func runKeepAlive(args []string) error {
	var scriptFlag bool
	var portFlag int
	args, err := flags.
		Bool("--script", &scriptFlag).
		Int("--port", &portFlag).
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
	return runKeepAliveLoop(port, args)
}

// runKeepAliveLoop implements the keep-alive logic in Go.
func runKeepAliveLoop(port int, serverArgs []string) error {
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
		fmt.Printf("[%s] Starting ai-critic server on port %d...\n", timestamp(), port)

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

		// Health check loop
		exitCode := healthCheckLoop(port, cmd)

		fmt.Printf("[%s] Server exited with code %d, restarting in %v...\n", timestamp(), exitCode, restartDelay)
		time.Sleep(restartDelay)
	}
}

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

// healthCheckLoop periodically checks port accessibility.
// Returns the exit code when the server exits or needs to be killed.
func healthCheckLoop(port int, cmd *exec.Cmd) int {
	// Channel to receive process exit
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			// Process exited on its own
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					return exitErr.ExitCode()
				}
				return 1
			}
			return 0
		case <-ticker.C:
			if !isPortReachable(port) {
				fmt.Printf("[%s] Port %d is not accessible, killing server (PID=%d)...\n", timestamp(), port, cmd.Process.Pid)
				killProcess(cmd)
				// Wait for the process to finish
				<-done
				return 1
			}
		}
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
		cmd.Process.Wait()
	}
}

func timestamp() string {
	return time.Now().Format("2006-01-02T15:04:05")
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
