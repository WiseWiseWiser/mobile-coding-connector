package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

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
	mux.HandleFunc("/api/keep-alive/upload-target", s.handleUploadTarget)
	mux.HandleFunc("/api/keep-alive/set-binary", s.handleSetBinary)
	mux.HandleFunc("/api/keep-alive/logs", s.handleLogs)
	mux.HandleFunc("/api/keep-alive/buildable-projects", s.handleBuildableProjects)
	mux.HandleFunc("/api/keep-alive/build-next", s.handleBuildNext)

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
	ServerPort          int    `json:"server_port"`
	ServerPID           int    `json:"server_pid"`
	KeepAlivePort       int    `json:"keep_alive_port"`
	KeepAlivePID        int    `json:"keep_alive_pid"`
	StartedAt           string `json:"started_at,omitempty"`
	Uptime              string `json:"uptime,omitempty"`
	NextBinary          string `json:"next_binary,omitempty"`
	NextHealthCheckTime string `json:"next_health_check_time,omitempty"`
}

func (s *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	snapshot := s.state.GetStatusSnapshot()

	resp := StatusResponse{
		Running:       snapshot.ServerPID > 0,
		BinaryPath:    snapshot.BinPath,
		ServerPort:    snapshot.ServerPort,
		ServerPID:     snapshot.ServerPID,
		KeepAlivePort: config.KeepAlivePort,
		KeepAlivePID:  os.Getpid(),
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

func (s *HTTPServer) handleBuildableProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	buildable, err := FindBuildableProjects()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to find buildable projects: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buildable)
}

func (s *HTTPServer) handleBuildNext(w http.ResponseWriter, r *http.Request) {
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

	// Delegate to build.go
	BuildNextBinary(w, r, req.ProjectID, s.state.GetBinPath())
}
