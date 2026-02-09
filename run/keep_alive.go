package run

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
	"github.com/xhd2015/lifelog-private/ai-critic/server/terminal"
)

const (
	startupTimeout       = 10 * time.Second
	healthCheckInterval  = 10 * time.Second
	restartDelay         = 3 * time.Second
	portCheckTimeout     = 2 * time.Second
	upgradeCheckInterval = 30 * time.Second
)

// keepAliveState holds the mutable state of the keep-alive daemon, guarded by mu.
type keepAliveState struct {
	mu         sync.Mutex
	binPath    string    // current binary being run
	serverPort int       // the port the managed server listens on
	serverPID  int       // PID of the currently running server, 0 if not running
	startedAt  time.Time // when the current server was started
	restartCh  chan struct{}
}

var kaState = &keepAliveState{
	restartCh: make(chan struct{}, 1),
}

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

	port := config.DefaultServerPort
	if portFlag > 0 {
		port = portFlag
	}

	if scriptFlag {
		return outputKeepAliveScript(port, args)
	}

	// Start the keep-alive management HTTP server
	go startKeepAliveHTTPServer()

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

	kaState.mu.Lock()
	kaState.binPath = binPath
	kaState.serverPort = port
	kaState.mu.Unlock()

	logFile, err := os.OpenFile(config.ServerLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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
		kaState.mu.Lock()
		currentBin := kaState.binPath
		kaState.mu.Unlock()

		newerBin := findNewerBinary(currentBin)
		if newerBin != "" {
			fmt.Printf("[%s] Found newer binary: %s (upgrading from %s)\n", timestamp(), newerBin, filepath.Base(currentBin))
			kaState.mu.Lock()
			kaState.binPath = newerBin
			currentBin = newerBin
			kaState.mu.Unlock()
		}

		fmt.Printf("[%s] Starting ai-critic server on port %d (binary: %s)...\n", timestamp(), port, filepath.Base(currentBin))

		// Build server args: include --port if it was specified
		cmdArgs := append([]string{}, serverArgs...)

		// Ensure the binary is executable
		os.Chmod(currentBin, 0755)

		cmd := exec.Command(currentBin, cmdArgs...)
		cmd.Dir, _ = os.Getwd()

		// Create a new process group so we can kill all child processes
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

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
			kaState.mu.Lock()
			kaState.serverPID = 0
			kaState.mu.Unlock()
			fmt.Printf("[%s] Restarting in %v...\n", timestamp(), restartDelay)
			time.Sleep(restartDelay)
			continue
		}

		pid := cmd.Process.Pid
		fmt.Printf("[%s] Server started (PID=%d)\n", timestamp(), pid)

		kaState.mu.Lock()
		kaState.serverPID = pid
		kaState.startedAt = time.Now()
		kaState.mu.Unlock()

		// Wait for port to become ready
		ready := waitForPort(port, startupTimeout, cmd)
		if !ready {
			fmt.Printf("[%s] ERROR: Server failed to become ready within %v\n", timestamp(), startupTimeout)
			killProcessGroup(cmd)
			kaState.mu.Lock()
			kaState.serverPID = 0
			kaState.mu.Unlock()
			fmt.Printf("[%s] Restarting in %v...\n", timestamp(), restartDelay)
			time.Sleep(restartDelay)
			continue
		}

		fmt.Printf("[%s] Server is ready (PID=%d, port=%d)\n", timestamp(), pid, port)

		// Health check loop (also checks for binary upgrades and restart signals)
		exitReason := healthCheckLoop(port, cmd, currentBin)

		kaState.mu.Lock()
		kaState.serverPID = 0
		kaState.mu.Unlock()

		switch exitReason {
		case exitReasonUpgrade, exitReasonRestart:
			fmt.Printf("[%s] %s, restarting immediately...\n", timestamp(), exitReason)
		default:
			fmt.Printf("[%s] Server exited (%s), restarting in %v...\n", timestamp(), exitReason, restartDelay)
			time.Sleep(restartDelay)
		}
	}
}

type exitReasonType string

const (
	exitReasonProcessExit exitReasonType = "process exited"
	exitReasonPortDead    exitReasonType = "port unreachable"
	exitReasonUpgrade     exitReasonType = "binary upgrade"
	exitReasonRestart     exitReasonType = "restart requested"
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
	// Channel to receive process exit.
	// We use cmd.Process.Wait() instead of cmd.Wait() to avoid blocking
	// on stdout/stderr pipe closure from child processes.
	done := make(chan struct{}, 1)
	go func() {
		cmd.Process.Wait()
		done <- struct{}{}
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

		case <-kaState.restartCh:
			// Restart requested via API
			fmt.Printf("[%s] Restart requested via API, stopping server (PID=%d)...\n", timestamp(), cmd.Process.Pid)
			gracefulStopGroup(cmd)
			waitForDone(done, 5*time.Second)
			return exitReasonRestart

		case <-healthTicker.C:
			if !isPortReachable(port) {
				consecutiveFailures++
				fmt.Printf("[%s] Port %d health check failed (%d/%d)\n", timestamp(), port, consecutiveFailures, maxConsecutiveFailures)

				if consecutiveFailures >= maxConsecutiveFailures {
					fmt.Printf("[%s] Port %d is not accessible after %d checks, killing server (PID=%d)...\n",
						timestamp(), port, consecutiveFailures, cmd.Process.Pid)
					killProcessGroup(cmd)
					waitForDone(done, 5*time.Second)
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
				kaState.mu.Lock()
				kaState.binPath = newerBin
				kaState.mu.Unlock()
				gracefulStopGroup(cmd)
				waitForDone(done, 5*time.Second)
				return exitReasonUpgrade
			}
		}
	}
}

// waitForDone waits for the done signal with a timeout.
func waitForDone(done <-chan struct{}, timeout time.Duration) {
	select {
	case <-done:
	case <-time.After(timeout):
	}
}

// gracefulStopGroup sends SIGTERM to the process group first, waits briefly, then SIGKILL.
func gracefulStopGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fallback: kill just the process
		cmd.Process.Signal(syscall.SIGTERM)
		time.Sleep(3 * time.Second)
		cmd.Process.Signal(syscall.SIGKILL)
		return
	}

	// Send SIGTERM to the entire process group
	syscall.Kill(-pgid, syscall.SIGTERM)

	// Wait up to 5 seconds for graceful shutdown
	time.Sleep(5 * time.Second)

	// Force kill the entire process group
	syscall.Kill(-pgid, syscall.SIGKILL)
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

// killProcessGroup kills the entire process group.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fallback: kill just the process
		cmd.Process.Signal(syscall.SIGKILL)
		return
	}
	syscall.Kill(-pgid, syscall.SIGKILL)
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

// ---- Keep-Alive Management HTTP Server ----

func startKeepAliveHTTPServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/keep-alive/status", handleKeepAliveStatus)
	mux.HandleFunc("/api/keep-alive/restart", handleKeepAliveRestart)
	mux.HandleFunc("/api/keep-alive/upload-target", handleKeepAliveUploadTarget)
	mux.HandleFunc("/api/keep-alive/set-binary", handleKeepAliveSetBinary)
	mux.HandleFunc("/api/keep-alive/logs", handleKeepAliveLogs)
	mux.HandleFunc("/api/keep-alive/buildable-projects", handleBuildableProjects)
	mux.HandleFunc("/api/keep-alive/build-next", handleBuildNext)

	addr := fmt.Sprintf(":%d", config.KeepAlivePort)
	fmt.Printf("[%s] Keep-alive management server listening on %s\n", timestamp(), addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("[%s] Keep-alive management server error: %v\n", timestamp(), err)
	}
}

type keepAliveStatusResponse struct {
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

func handleKeepAliveStatus(w http.ResponseWriter, r *http.Request) {
	kaState.mu.Lock()
	currentBin := kaState.binPath
	resp := keepAliveStatusResponse{
		Running:       kaState.serverPID > 0,
		BinaryPath:    currentBin,
		ServerPort:    kaState.serverPort,
		ServerPID:     kaState.serverPID,
		KeepAlivePort: config.KeepAlivePort,
		KeepAlivePID:  os.Getpid(),
	}
	if kaState.serverPID > 0 && !kaState.startedAt.IsZero() {
		resp.StartedAt = kaState.startedAt.Format(time.RFC3339)
		resp.Uptime = time.Since(kaState.startedAt).Truncate(time.Second).String()
	}
	kaState.mu.Unlock()

	// Check for newer binary
	if newerBin := findNewerBinary(currentBin); newerBin != "" {
		resp.NextBinary = newerBin
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleKeepAliveRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Non-blocking send to restart channel
	select {
	case kaState.restartCh <- struct{}{}:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "restart_requested"})
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "restart_already_pending"})
	}
}

// handleKeepAliveUploadTarget returns the path where the next binary should be uploaded.
// GET /api/keep-alive/upload-target
func handleKeepAliveUploadTarget(w http.ResponseWriter, r *http.Request) {
	kaState.mu.Lock()
	currentBin := kaState.binPath
	kaState.mu.Unlock()

	dir := filepath.Dir(currentBin)
	currentBase, currentVersion := parseBinVersion(currentBin)
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

// handleKeepAliveSetBinary notifies the keep-alive daemon that a new binary has been uploaded
// and is ready to use. It makes the binary executable.
// POST /api/keep-alive/set-binary  { "path": "/path/to/binary" }
func handleKeepAliveSetBinary(w http.ResponseWriter, r *http.Request) {
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

	fmt.Printf("[%s] New binary set: %s (%d bytes)\n", timestamp(), req.Path, info.Size())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"path":   req.Path,
		"size":   info.Size(),
	})
}

// handleKeepAliveLogs streams the server log via tail -fn100, using the shared SSE writer.
// GET /api/keep-alive/logs?lines=100
func handleKeepAliveLogs(w http.ResponseWriter, r *http.Request) {
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	linesStr := r.URL.Query().Get("lines")
	maxLines := "100"
	if linesStr != "" {
		if n, err := strconv.Atoi(linesStr); err == nil && n > 0 {
			maxLines = strconv.Itoa(n)
		}
	}

	logPath := config.ServerLogFile
	cmd := exec.Command("tail", "-fn"+maxLines, logPath)

	// Kill tail when the client disconnects
	ctx := r.Context()
	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}()

	if err := sw.StreamCmd(cmd); err != nil {
		sw.SendError(fmt.Sprintf("tail error: %v", err))
	}
	// tail -f runs indefinitely until killed, so no done event needed.
	// When client disconnects, context is cancelled and tail is killed.
}

// ---- Build from Source ----

// buildableProject represents a project that can be built
type buildableProject struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Dir            string `json:"dir"`
	HasGoMod       bool   `json:"has_go_mod"`
	HasBuildScript bool   `json:"has_build_script"`
}

// findBuildableProjects scans all projects and finds those that can be built.
// A project is buildable if it has go.mod and script/server/build/for-linux-amd64
func findBuildableProjects() ([]buildableProject, error) {
	projectsFile := config.ProjectsFile

	data, err := os.ReadFile(projectsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []buildableProject{}, nil
		}
		return nil, err
	}

	var projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Dir  string `json:"dir"`
	}
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}

	var buildable []buildableProject
	for _, p := range projects {
		if p.Dir == "" {
			continue
		}

		// Check if directory exists
		info, err := os.Stat(p.Dir)
		if err != nil || !info.IsDir() {
			continue
		}

		// Check for go.mod
		hasGoMod := false
		if _, err := os.Stat(filepath.Join(p.Dir, "go.mod")); err == nil {
			hasGoMod = true
		}

		// Check for build script
		hasBuildScript := false
		buildScriptPath := filepath.Join(p.Dir, "script", "server", "build", "for-linux-amd64")
		if _, err := os.Stat(buildScriptPath); err == nil {
			hasBuildScript = true
		}

		if hasGoMod && hasBuildScript {
			buildable = append(buildable, buildableProject{
				ID:             p.ID,
				Name:           p.Name,
				Dir:            p.Dir,
				HasGoMod:       true,
				HasBuildScript: true,
			})
		}
	}

	return buildable, nil
}

// handleBuildableProjects returns the list of projects that can be built from source.
// GET /api/keep-alive/buildable-projects
func handleBuildableProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	buildable, err := findBuildableProjects()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to find buildable projects: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buildable)
}

// handleBuildNext builds the next binary from a project source with SSE streaming.
// POST /api/keep-alive/build-next
// Request: { "project_id": "..." }
// Response: SSE stream with build output, then done event with result
func handleBuildNext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Find buildable projects
	buildable, err := findBuildableProjects()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to find buildable projects: %v", err), http.StatusInternalServerError)
		return
	}

	// Find the requested project or use the first available
	var project *buildableProject
	if req.ProjectID != "" {
		for i := range buildable {
			if buildable[i].ID == req.ProjectID {
				project = &buildable[i]
				break
			}
		}
	} else if len(buildable) > 0 {
		project = &buildable[0]
	}

	if project == nil {
		http.Error(w, "no buildable project found", http.StatusBadRequest)
		return
	}

	// Get the upload target path (next binary)
	kaState.mu.Lock()
	currentBin := kaState.binPath
	kaState.mu.Unlock()

	dir := filepath.Dir(currentBin)
	currentBase, currentVersion := parseBinVersion(currentBin)
	nextVersion := currentVersion + 1
	newName := fmt.Sprintf("%s-v%d", currentBase, nextVersion)
	destPath := filepath.Join(dir, newName)

	// Create SSE writer
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Log build start
	sw.SendLog(fmt.Sprintf("Building next binary (v%d) from project %s...", nextVersion, project.Name))
	sw.SendLog(fmt.Sprintf("Target: %s", destPath))

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		sw.SendError(fmt.Sprintf("Failed to create destination directory: %v", err))
		return
	}

	// Run build script
	buildScriptPath := filepath.Join(project.Dir, "script", "server", "build", "for-linux-amd64")
	cmd := exec.Command(buildScriptPath, "-o", destPath)
	cmd.Dir = project.Dir

	err = sw.StreamCmd(cmd)
	if err != nil {
		sw.SendError(fmt.Sprintf("Build failed: %v", err))
		return
	}

	// Make binary executable
	if err := os.Chmod(destPath, 0755); err != nil {
		sw.SendError(fmt.Sprintf("Failed to chmod binary: %v", err))
		return
	}

	// Get file size
	info, err := os.Stat(destPath)
	if err != nil {
		sw.SendError(fmt.Sprintf("Failed to stat binary: %v", err))
		return
	}

	// Log success
	sw.SendLog(fmt.Sprintf("Build successful: %s (%d bytes)", destPath, info.Size()))

	// Send done event with result data
	sw.SendDone(map[string]string{
		"success":      "true",
		"message":      fmt.Sprintf("Built %s (%s) v%d", newName, project.Name, nextVersion),
		"binary_path":  destPath,
		"binary_name":  newName,
		"version":      strconv.Itoa(nextVersion),
		"size":         strconv.FormatInt(info.Size(), 10),
		"project_name": project.Name,
	})
}

// ---- Keep-Alive Request Command ----

// runKeepAliveRequest sends request commands to a running keep-alive daemon.
// Usage: ai-critic keep-alive request <action>
// Actions: info, restart
func runKeepAliveRequest(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("action required: info, restart")
	}

	action := args[0]

	switch action {
	case "info":
		return sendKeepAliveInfo()
	case "restart":
		return sendKeepAliveRestart()
	default:
		return fmt.Errorf("unknown action: %s (use 'info' or 'restart')", action)
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
