package run

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/run/daemon"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/terminal"
)

const keepAliveHelp = `Usage: ai-critic keep-alive [options]
       ai-critic keep-alive status [options]
       ai-critic keep-alive logs [options]
       ai-critic keep-alive exec-replace <new-binary> [options]

Keep the ai-critic server running with automatic restart and health checking.

Options:
  --port PORT         Port to run the server on (default: %d)
  --forever           Skip port-in-use check and start anyway
  --log FILE          Log keep-alive output to file (default: ai-critic-server-keep-alive.log)
                      Use --log no to disable logging to file
  --script            Output shell script instead of running Go code
  -h, --help          Show this help message

Commands:
  ai-critic keep-alive status             Get daemon status
  ai-critic keep-alive logs               Follow keep-alive daemon logs (tail -fn100)
  ai-critic keep-alive exec-replace FILE  Replace keep-alive daemon binary via exec
  ai-critic keep-alive request info       Get daemon status
  ai-critic keep-alive request status     Get daemon status
  ai-critic keep-alive request restart    Request server restart
  ai-critic keep-alive request fix-tunnel Fix stale Cloudflare tunnel
`

const defaultKeepAliveLogPath = "ai-critic-server-keep-alive.log"

func runKeepAlive(args []string) error {
	var scriptFlag bool
	var portFlag int
	var foreverFlag bool
	var logFlag string

	args, err := flags.
		Bool("--script", &scriptFlag).
		Int("--port", &portFlag).
		Bool("--forever", &foreverFlag).
		String("--log", &logFlag).
		Help("-h,--help", fmt.Sprintf(keepAliveHelp, config.DefaultServerPort)).
		Parse(args)
	if err != nil {
		return err
	}

	port := config.DefaultServerPort
	if portFlag > 0 {
		port = portFlag
	}

	if scriptFlag {
		return outputKeepAliveScript(port, args)
	}

	logPath := resolveKeepAliveLogPath(logFlag)

	if len(args) > 0 {
		switch args[0] {
		case "status":
			if len(args) > 1 {
				return fmt.Errorf("keep-alive status does not accept extra args: %s", strings.Join(args[1:], " "))
			}
			return sendKeepAliveInfo()
		case "logs":
			if len(args) > 1 {
				return fmt.Errorf("keep-alive logs does not accept extra args: %s", strings.Join(args[1:], " "))
			}
			return followKeepAliveLogs(logPath)
		case "exec-replace":
			if len(args) != 2 {
				return fmt.Errorf("usage: ai-critic keep-alive exec-replace <new-binary>")
			}
			return sendKeepAliveExecReplace(args[1])
		}
	}

	return daemon.RunKeepAlive(port, foreverFlag, logPath, args)
}

func resolveKeepAliveLogPath(logFlag string) string {
	if logFlag == "no" {
		return ""
	}
	if logFlag != "" {
		return logFlag
	}
	return defaultKeepAliveLogPath
}

// runKeepAliveRequest sends request commands to a running keep-alive daemon.
func runKeepAliveRequest(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("action required: info, status, restart, fix-tunnel")
	}

	action := args[0]

	switch action {
	case "info":
		return sendKeepAliveInfo()
	case "status":
		return sendKeepAliveInfo()
	case "restart":
		return sendKeepAliveRestart()
	case "fix-tunnel":
		return sendKeepAliveFixTunnel()
	default:
		return fmt.Errorf("unknown action: %s (use 'info', 'status', 'restart', or 'fix-tunnel')", action)
	}
}

// sendKeepAliveInfo fetches and displays status from the keep-alive daemon.
func sendKeepAliveInfo() error {
	url := fmt.Sprintf("http://localhost:%d/api/keep-alive/status", config.KeepAlivePort)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to keep-alive daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("keep-alive daemon returned error: %s", resp.Status)
	}

	var status struct {
		Running       bool   `json:"running"`
		BinaryPath    string `json:"binary_path"`
		ServerPort    int    `json:"server_port"`
		ServerPID     int    `json:"server_pid"`
		KeepAlivePort int    `json:"keep_alive_port"`
		KeepAlivePID  int    `json:"keep_alive_pid"`
		StartedAt     string `json:"started_at,omitempty"`
		Uptime        string `json:"uptime,omitempty"`
		NextBinary    string `json:"next_binary,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	fmt.Println("Keep-Alive Daemon Status")
	fmt.Println("========================")
	fmt.Printf("Running:        %v\n", status.Running)
	fmt.Printf("Server PID:     %d\n", status.ServerPID)
	fmt.Printf("Server Port:    %d\n", status.ServerPort)
	fmt.Printf("Keep-Alive PID: %d\n", status.KeepAlivePID)
	fmt.Printf("Keep-Alive Port:%d\n", status.KeepAlivePort)
	fmt.Printf("Binary Path:    %s\n", status.BinaryPath)
	if status.Uptime != "" {
		fmt.Printf("Uptime:         %s\n", status.Uptime)
	}
	if status.StartedAt != "" {
		fmt.Printf("Started At:     %s\n", status.StartedAt)
	}
	if status.NextBinary != "" {
		fmt.Printf("Next Binary:    %s\n", status.NextBinary)
	}

	return nil
}

func followKeepAliveLogs(logPath string) error {
	if logPath == "" {
		return fmt.Errorf("keep-alive log file is disabled (--log no), nothing to follow")
	}

	cmd := exec.Command("tail", "-fn100", logPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tail keep-alive logs %q: %v", logPath, err)
	}
	return nil
}

func sendKeepAliveExecReplace(newBinary string) error {
	absPath, err := filepath.Abs(newBinary)
	if err != nil {
		return fmt.Errorf("resolve binary path: %v", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat binary %q: %v", absPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("binary path %q is a directory", absPath)
	}

	url := fmt.Sprintf("http://localhost:%d/api/keep-alive/exec-replace", config.KeepAlivePort)
	reqBody := strings.NewReader(fmt.Sprintf(`{"binary_path":%q}`, absPath))

	resp, err := http.Post(url, "application/json", reqBody)
	if err != nil {
		return fmt.Errorf("failed to connect to keep-alive daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("keep-alive daemon returned error: %s", resp.Status)
	}

	var result struct {
		Status     string `json:"status"`
		BinaryPath string `json:"binary_path"`
		Message    string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if result.Message != "" {
		fmt.Println(result.Message)
		return nil
	}
	fmt.Printf("Exec-replace requested: %s\n", result.BinaryPath)
	return nil
}

// sendKeepAliveRestart sends a restart request to the keep-alive daemon.
func sendKeepAliveRestart() error {
	url := fmt.Sprintf("http://localhost:%d/api/keep-alive/restart", config.KeepAlivePort)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to keep-alive daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("keep-alive daemon returned error: %s", resp.Status)
	}

	var result struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	fmt.Printf("Restart request sent: %s\n", result.Status)
	return nil
}

// sendKeepAliveFixTunnel sends a request to fix stale tunnels to the keep-alive daemon.
func sendKeepAliveFixTunnel() error {
	url := fmt.Sprintf("http://localhost:%d/api/keep-alive/fix-tunnel", config.KeepAlivePort)

	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to keep-alive daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("keep-alive daemon returned error: %s", resp.Status)
	}

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message,omitempty"`
		Fixed   int    `json:"fixed,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	fmt.Printf("Fix tunnel request: %s\n", result.Status)
	if result.Message != "" {
		fmt.Printf("Message: %s\n", result.Message)
	}
	if result.Fixed > 0 {
		fmt.Printf("Fixed %d tunnel(s)\n", result.Fixed)
	}
	return nil
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
	if port != config.DefaultServerPort {
		cmdParts = append(cmdParts, "--port", fmt.Sprintf("%d", port))
	}
	for _, a := range serverArgs {
		cmdParts = append(cmdParts, terminal.ShellQuote(a))
	}
	serverCmd := strings.Join(cmdParts, " ")

	script := fmt.Sprintf(`#!/bin/sh
LOG_FILE="%s"
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
`, config.ServerLogFile, port, terminal.ShellQuote(binPath), serverCmd, serverCmd)

	fmt.Print(script)
	return nil
}

// timestamp returns a formatted timestamp string for logging
func timestamp() string {
	return time.Now().Format("2006-01-02T15:04:05")
}
