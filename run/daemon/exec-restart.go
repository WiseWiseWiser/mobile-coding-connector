package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

// callExecRestartEndpoint calls the server's /api/server/exec-restart endpoint
// which performs a graceful shutdown and then uses syscall.Exec to replace
// the current process with the new binary (preserving PID).
// Returns true if the request was successful (the server will exec and not return).
func callExecRestartEndpoint() bool {
	token, err := loadFirstToken()
	if err != nil {
		Logger("Failed to load auth token: %v", err)
		return false
	}

	port := config.DefaultServerPort
	url := fmt.Sprintf("http://localhost:%d/api/server/exec-restart", port)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		Logger("Failed to create exec-restart request: %v", err)
		return false
	}

	if token != "" {
		req.AddCookie(&http.Cookie{
			Name:  "ai-critic-token",
			Value: token,
		})
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		Logger("Failed to call exec-restart endpoint: %v", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		Logger("Exec-restart request accepted (server will restart)")
		return true
	}

	Logger("Exec-restart endpoint returned status: %d", resp.StatusCode)
	return false
}

func (s *HTTPServer) handleExecReplace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		BinaryPath string `json:"binary_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}
	if req.BinaryPath == "" {
		http.Error(w, "binary_path is required", http.StatusBadRequest)
		return
	}

	binaryPath, err := filepath.Abs(req.BinaryPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("resolve binary path: %v", err), http.StatusBadRequest)
		return
	}
	if err := validateExecutableReplacement(binaryPath); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.shutdownDaemonForExec(false, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := triggerDaemonExecReplace(w, binaryPath, false); err != nil {
		s.state.ClearDaemonShutdown()
		Logger("ERROR: daemon exec-replace request failed for %s: %v", binaryPath, err)
	}
}

// handleRestartDaemon handles restarting the keep-alive daemon itself using exec.
// It streams logs via SSE, finds the newest binary, and replaces the current process.
func (s *HTTPServer) handleRestartDaemon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.performDaemonRestartDaemon(w); err != nil {
		Logger("ERROR: restart-daemon request failed: %v", err)
	}
}

func (s *HTTPServer) performDaemonRestartDaemon(w http.ResponseWriter) error {
	sseWriter := sse.NewWriter(w)
	if sseWriter == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return fmt.Errorf("streaming not supported")
	}

	sseWriter.SendLog("Restarting keep-alive daemon...")

	shutdownPrepared := false
	defer func() {
		if shutdownPrepared {
			s.state.ClearDaemonShutdown()
		}
	}()

	if err := s.shutdownDaemonForExec(true, sseWriter); err != nil {
		sseWriter.SendError(err.Error())
		sseWriter.SendDone(map[string]string{"success": "false"})
		return err
	}
	shutdownPrepared = true

	currentBin, err := os.Executable()
	if err != nil {
		sseWriter.SendError(fmt.Sprintf("Failed to get current executable: %v", err))
		sseWriter.SendDone(map[string]string{"success": "false"})
		return err
	}
	sseWriter.SendLog(fmt.Sprintf("Current binary: %s", currentBin))

	newerBin := FindNewerBinary(currentBin)
	if newerBin == "" {
		sseWriter.SendLog("No newer binary found, using current binary")
		newerBin = currentBin
	} else {
		sseWriter.SendLog(fmt.Sprintf("Found newer binary: %s", newerBin))
	}

	workDir, err := os.Getwd()
	if err != nil {
		sseWriter.SendError(fmt.Sprintf("Failed to get working directory: %v", err))
		sseWriter.SendDone(map[string]string{"success": "false"})
		return err
	}
	sseWriter.SendLog(fmt.Sprintf("Working directory: %s", workDir))

	args := buildDaemonExecArgs(newerBin)
	sseWriter.SendLog(fmt.Sprintf("Arguments: %v", args))

	if err := os.Chmod(newerBin, 0755); err != nil {
		sseWriter.SendError(fmt.Sprintf("Failed to make binary executable: %v", err))
		sseWriter.SendDone(map[string]string{"success": "false"})
		return err
	}

	sseWriter.SendLog("Preparing to exec...")
	sseWriter.SendStatus("restarting", map[string]string{
		"binary": newerBin,
		"args":   fmt.Sprintf("%v", args),
	})

	sseWriter.SendDone(map[string]string{
		"success":   "true",
		"message":   "Daemon restarting via exec",
		"binary":    newerBin,
		"directory": workDir,
	})

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	time.Sleep(100 * time.Millisecond)

	shutdownPrepared = false
	return execReplaceCurrentProcess(newerBin, args, os.Environ())
}

func validateExecutableReplacement(binaryPath string) error {
	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("binary not found: %v", err)
	}
	if info.IsDir() {
		return fmt.Errorf("binary path is a directory: %s", binaryPath)
	}
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %v", err)
	}
	return nil
}

func triggerDaemonExecReplace(w http.ResponseWriter, binaryPath string, stopManagedServer bool) error {
	Logger("Preparing daemon exec-replace (binary=%s, stopManagedServer=%v)", binaryPath, stopManagedServer)

	args := buildDaemonExecArgs(binaryPath)
	env := os.Environ()
	if !stopManagedServer {
		env = withEnvValue(env, keepAliveSkipServerPortCheckEnv, "1")
	}

	response := map[string]string{
		"status":      "ok",
		"binary_path": binaryPath,
		"message":     fmt.Sprintf("Keep-alive daemon exec-replace requested: %s", binaryPath),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	time.Sleep(100 * time.Millisecond)

	return execReplaceCurrentProcess(binaryPath, args, env)
}

func buildDaemonExecArgs(binaryPath string) []string {
	return append([]string{binaryPath}, os.Args[1:]...)
}

func execReplaceCurrentProcess(binaryPath string, args []string, env []string) error {
	Logger("Executing daemon replacement: %s %v", binaryPath, args)
	err := syscall.Exec(binaryPath, args, env)
	Logger("ERROR: syscall.Exec failed: %v", err)
	GlobalState.ClearDaemonShutdown()
	return err
}

func withEnvValue(env []string, key string, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			envCopy := append([]string(nil), env...)
			envCopy[i] = prefix + value
			return envCopy
		}
	}
	return append(append([]string(nil), env...), prefix+value)
}
