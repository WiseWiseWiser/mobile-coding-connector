package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

var currentCmd atomic.Value

// setCurrentCommand stores the current server command (thread-safe)
func setCurrentCommand(cmd *exec.Cmd) {
	currentCmd.Store(cmd)
}

// getCurrentCommand retrieves the current server command (thread-safe)
func getCurrentCommand() *exec.Cmd {
	val := currentCmd.Load()
	if val == nil {
		return nil
	}
	return val.(*exec.Cmd)
}

// HTTPServer provides the HTTP management API for the keep-alive daemon
type HTTPServer struct {
	state *State
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(state *State) *HTTPServer {
	return &HTTPServer{
		state: state,
	}
}

// Start starts the HTTP management server in a goroutine
func (s *HTTPServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/keep-alive/status", s.handleStatus)
	mux.HandleFunc("/api/keep-alive/restart", s.handleRestart)
	mux.HandleFunc("/api/keep-alive/fix-tunnel", s.handleFixTunnel)
	mux.HandleFunc("/api/keep-alive/upload-target", s.handleUploadTarget)
	mux.HandleFunc("/api/keep-alive/set-binary", s.handleSetBinary)
	mux.HandleFunc("/api/keep-alive/logs", s.handleLogs)
	mux.HandleFunc("/api/keep-alive/restart-daemon", s.handleRestartDaemon)

	addr := fmt.Sprintf(":%d", config.KeepAlivePort)
	Logger("Keep-alive management server listening on %s", addr)

	go func() {
		if err := http.ListenAndServe(addr, mux); err != nil {
			Logger("Keep-alive management server error: %v", err)
		}
	}()
}

// StatusResponse represents the daemon status JSON response
type StatusResponse struct {
	Running             bool   `json:"running"`
	BinaryPath          string `json:"binary_path"`
	DaemonBinaryPath    string `json:"daemon_binary_path"`
	ServerPort          int    `json:"server_port"`
	ServerPID           int    `json:"server_pid"`
	KeepAlivePort       int    `json:"keep_alive_port"`
	KeepAlivePID        int    `json:"keep_alive_pid"`
	StartedAt           string `json:"started_at,omitempty"`
	Uptime              string `json:"uptime,omitempty"`
	NextBinary          string `json:"next_binary,omitempty"`
	NextHealthCheckTime string `json:"next_health_check_time,omitempty"`
	RestartCount        int    `json:"restart_count"`
}

func (s *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	snapshot := s.state.GetStatusSnapshot()

	resp := StatusResponse{
		Running:          snapshot.ServerPID > 0,
		BinaryPath:       snapshot.BinPath,
		DaemonBinaryPath: snapshot.DaemonBinPath,
		ServerPort:       snapshot.ServerPort,
		ServerPID:        snapshot.ServerPID,
		KeepAlivePort:    config.KeepAlivePort,
		KeepAlivePID:     os.Getpid(),
		RestartCount:     snapshot.RestartCount,
	}

	if snapshot.ServerPID > 0 && !snapshot.StartedAt.IsZero() {
		resp.StartedAt = snapshot.StartedAt.Format(time.RFC3339)
		resp.Uptime = time.Since(snapshot.StartedAt).Truncate(time.Second).String()
	}

	if !snapshot.NextHealthCheckTime.IsZero() {
		resp.NextHealthCheckTime = snapshot.NextHealthCheckTime.Format(time.RFC3339)
	}

	// Check for newer binary
	if newerBin := FindNewerBinary(snapshot.BinPath); newerBin != "" {
		resp.NextBinary = newerBin
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *HTTPServer) handleRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Non-blocking send to restart channel
	if s.state.RequestRestart() {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "restart_requested"})
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "restart_already_pending"})
	}
}

func (s *HTTPServer) handleFixTunnel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the server port from state
	snapshot := s.state.GetStatusSnapshot()
	if snapshot.ServerPort == 0 {
		http.Error(w, "server port not available", http.StatusServiceUnavailable)
		return
	}

	// Perform the tunnel fix operation
	result := s.fixStaleTunnels(snapshot.ServerPort)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *HTTPServer) handleUploadTarget(w http.ResponseWriter, r *http.Request) {
	snapshot := s.state.GetStatusSnapshot()
	currentBin := snapshot.BinPath

	dir := filepath.Dir(currentBin)
	currentBase, currentVersion := ParseBinVersion(currentBin)
	nextVersion := currentVersion + 1
	newName := fmt.Sprintf("%s-v%d", currentBase, nextVersion)
	destPath := filepath.Join(dir, newName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"path":            destPath,
		"binary_name":     newName,
		"current_version": currentVersion,
		"next_version":    nextVersion,
	})
}

func (s *HTTPServer) handleSetBinary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	// Verify the file exists
	info, err := os.Stat(req.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("binary not found: %v", err), http.StatusBadRequest)
		return
	}

	// Make executable
	if err := os.Chmod(req.Path, 0755); err != nil {
		http.Error(w, fmt.Sprintf("failed to chmod: %v", err), http.StatusInternalServerError)
		return
	}

	Logger("New binary set: %s (%d bytes)", req.Path, info.Size())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"path":   req.Path,
		"size":   info.Size(),
	})
}

func (s *HTTPServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	// Delegate to build.go for log streaming
	StreamLogs(w, r)
}

// fixStaleTunnels checks and fixes stale Cloudflare tunnels for the main server.
// It queries the main server's domains API and ensures DNS records point to the active tunnel.
func (s *HTTPServer) fixStaleTunnels(serverPort int) map[string]interface{} {
	result := map[string]interface{}{
		"status":  "checking",
		"fixed":   0,
		"message": "",
	}

	// Get the auth token from credentials file
	token, err := s.getAuthToken()
	if err != nil {
		result["status"] = "error"
		result["message"] = fmt.Sprintf("failed to get auth token: %v", err)
		return result
	}

	// Get domains configuration from main server
	domainsURL := fmt.Sprintf("http://localhost:%d/api/domains", serverPort)
	req, err := http.NewRequest("GET", domainsURL, nil)
	if err != nil {
		result["status"] = "error"
		result["message"] = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	// Add auth cookie
	req.AddCookie(&http.Cookie{
		Name:  "ai-critic-token",
		Value: token,
	})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result["status"] = "error"
		result["message"] = fmt.Sprintf("failed to connect to main server: %v", err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result["status"] = "error"
		result["message"] = fmt.Sprintf("main server returned status: %s", resp.Status)
		return result
	}

	var domainsResp struct {
		Domains []struct {
			Domain   string `json:"domain"`
			Provider string `json:"provider"`
			Status   string `json:"status"`
		} `json:"domains"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&domainsResp); err != nil {
		result["status"] = "error"
		result["message"] = fmt.Sprintf("failed to decode domains response: %v", err)
		return result
	}

	// Check if there are any cloudflare domains
	cloudflareDomains := []string{}
	for _, d := range domainsResp.Domains {
		if d.Provider == "cloudflare" {
			cloudflareDomains = append(cloudflareDomains, d.Domain)
		}
	}

	if len(cloudflareDomains) == 0 {
		result["status"] = "ok"
		result["message"] = "no cloudflare domains configured"
		return result
	}

	// For each domain, check if the DNS record points to the correct tunnel
	fixed := 0
	for _, domain := range cloudflareDomains {
		if s.fixDomainTunnel(domain, serverPort, token) {
			fixed++
		}
	}

	result["status"] = "ok"
	result["fixed"] = fixed
	if fixed > 0 {
		result["message"] = fmt.Sprintf("fixed %d tunnel(s)", fixed)
	} else {
		result["message"] = "all tunnels are healthy"
	}

	return result
}

// getAuthToken reads the server credentials file to get the auth token.
func (s *HTTPServer) getAuthToken() (string, error) {
	// Try to find the credentials file
	candidates := []string{
		"/root/.ai-critic/server-credentials",
		"/root/.config/ai-critic/server-credentials",
		filepath.Join(os.Getenv("HOME"), ".ai-critic/server-credentials"),
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return strings.TrimSpace(string(data)), nil
		}
	}

	return "", fmt.Errorf("credentials file not found")
}

// fixDomainTunnel fixes a single domain's tunnel by restarting it.
// Returns true if the tunnel was fixed.
func (s *HTTPServer) fixDomainTunnel(domain string, serverPort int, token string) bool {
	// Request the main server to restart the tunnel for this domain
	restartURL := fmt.Sprintf("http://localhost:%d/api/domains/tunnel/stop", serverPort)

	reqBody := fmt.Sprintf(`{"domain":"%s"}`, domain)
	req, err := http.NewRequest("POST", restartURL, strings.NewReader(reqBody))
	if err != nil {
		Logger("Failed to create stop request for %s: %v", domain, err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "ai-critic-token",
		Value: token,
	})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		Logger("Failed to stop tunnel for %s: %v", domain, err)
		// Continue anyway - it might not be running
	} else {
		resp.Body.Close()
	}

	// Now start the tunnel again
	startURL := fmt.Sprintf("http://localhost:%d/api/domains/tunnel/start", serverPort)
	req, err = http.NewRequest("POST", startURL, strings.NewReader(reqBody))
	if err != nil {
		Logger("Failed to create start request for %s: %v", domain, err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "ai-critic-token",
		Value: token,
	})

	resp, err = client.Do(req)
	if err != nil {
		Logger("Failed to start tunnel for %s: %v", domain, err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		Logger("Fixed tunnel for domain: %s", domain)
		return true
	}

	Logger("Failed to fix tunnel for %s: status %s", domain, resp.Status)
	return false
}

// killProcess kills a process by PID
func killProcess(cmd *exec.Cmd) {
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

	// Force kill the entire process group
	Logger("Killing process group %d", pgid)
	syscall.Kill(-pgid, syscall.SIGKILL)
}

// handleRestartDaemon handles restarting the keep-alive daemon itself using exec.
// It streams logs via SSE, finds the newest binary, and replaces the current process.
func (s *HTTPServer) handleRestartDaemon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sseWriter := sse.NewWriter(w)
	if sseWriter == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	sseWriter.SendLog("Restarting keep-alive daemon...")

	// Request daemon restart to stop health checker
	s.state.RequestDaemonRestart()
	sseWriter.SendLog("Health checker stop requested, waiting for it to exit...")
	time.Sleep(2 * time.Second)
	sseWriter.SendLog("Health checker should be stopped now")

	// Stop the running server gracefully before exec
	cmd := getCurrentCommand()
	if cmd != nil && cmd.Process != nil {
		pid := cmd.Process.Pid
		sseWriter.SendLog(fmt.Sprintf("Stopping server PID %d before restart...", pid))

		// Create a channel to wait for process exit
		done := make(chan struct{}, 1)
		go func() {
			cmd.Process.Wait()
			close(done)
		}()

		// Try graceful shutdown first
		if CallShutdownEndpoint() {
			sseWriter.SendLog("Graceful shutdown request sent")
			select {
			case <-done:
				sseWriter.SendLog("Server stopped gracefully")
			case <-time.After(30 * time.Second):
				sseWriter.SendLog("Graceful shutdown timeout, force killing...")
				killProcess(cmd)
				// Wait a bit for the process to actually die
				select {
				case <-done:
					sseWriter.SendLog("Server force stopped")
				case <-time.After(5 * time.Second):
					sseWriter.SendLog("Warning: server may still be running")
				}
			}
		} else {
			sseWriter.SendLog("Shutdown endpoint unavailable, using direct kill")
			killProcess(cmd)
			select {
			case <-done:
				sseWriter.SendLog("Server stopped")
			case <-time.After(5 * time.Second):
				sseWriter.SendLog("Warning: server may still be running")
			}
		}

		setCurrentCommand(nil)
		s.state.SetServerPID(0)
	}

	// Get current binary and args
	currentBin, err := os.Executable()
	if err != nil {
		sseWriter.SendError(fmt.Sprintf("Failed to get current executable: %v", err))
		sseWriter.SendDone(map[string]string{"success": "false"})
		return
	}
	sseWriter.SendLog(fmt.Sprintf("Current binary: %s", currentBin))

	// Find newest binary
	newerBin := FindNewerBinary(currentBin)
	if newerBin == "" {
		sseWriter.SendLog("No newer binary found, using current binary")
		newerBin = currentBin
	} else {
		sseWriter.SendLog(fmt.Sprintf("Found newer binary: %s", newerBin))
	}

	// Get current working directory
	workDir, err := os.Getwd()
	if err != nil {
		sseWriter.SendError(fmt.Sprintf("Failed to get working directory: %v", err))
		sseWriter.SendDone(map[string]string{"success": "false"})
		return
	}
	sseWriter.SendLog(fmt.Sprintf("Working directory: %s", workDir))

	// Get OS args (skip the first arg which is the program name)
	args := os.Args
	sseWriter.SendLog(fmt.Sprintf("Arguments: %v", args))

	// Ensure binary is executable
	if err := os.Chmod(newerBin, 0755); err != nil {
		sseWriter.SendError(fmt.Sprintf("Failed to make binary executable: %v", err))
		sseWriter.SendDone(map[string]string{"success": "false"})
		return
	}

	sseWriter.SendLog("Preparing to exec...")
	sseWriter.SendStatus("restarting", map[string]string{
		"binary": newerBin,
		"args":   fmt.Sprintf("%v", args),
	})

	// Send done before exec since exec won't return
	sseWriter.SendDone(map[string]string{
		"success":   "true",
		"message":   "Daemon restarting via exec",
		"binary":    newerBin,
		"directory": workDir,
	})

	// Flush to ensure client receives the done event
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Small delay to allow SSE to be sent
	time.Sleep(100 * time.Millisecond)

	// Execute the new binary, replacing current process
	// syscall.Exec never returns on success
	Logger("Executing: %s %v in %s", newerBin, args, workDir)
	err = syscall.Exec(newerBin, args, os.Environ())

	// If we get here, exec failed
	// We can't send SSE anymore since we've already sent done, so just log
	Logger("ERROR: syscall.Exec failed: %v", err)
}
