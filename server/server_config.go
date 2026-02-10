package server

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

// ServerConfigResponse represents the server configuration response
type ServerConfigResponse struct {
	ProjectDir       string `json:"project_dir"`
	AutoDetectedDir  string `json:"auto_detected_dir"`
	UsingExplicitDir bool   `json:"using_explicit_dir"`
}

// GetServerConfig handles GET /api/server/config
func GetServerConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get auto-detected directory (current working directory or --dir flag)
	autoDetectedDir := GetInitialDir()
	if autoDetectedDir == "" {
		var err error
		autoDetectedDir, err = os.Getwd()
		if err != nil {
			http.Error(w, "Failed to get working directory", http.StatusInternalServerError)
			return
		}
	}

	// Get explicit project dir from server-project.json
	serverProjectCfg, err := config.LoadServerProjectConfig()
	if err != nil {
		serverProjectCfg = &config.ServerProjectConfig{}
	}

	resp := ServerConfigResponse{
		ProjectDir:       serverProjectCfg.ProjectDir,
		AutoDetectedDir:  autoDetectedDir,
		UsingExplicitDir: serverProjectCfg.ProjectDir != "",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// SetServerConfigRequest represents the request to update server config
type SetServerConfigRequest struct {
	ProjectDir string `json:"project_dir"`
}

// SetServerConfig handles POST /api/server/config
func SetServerConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SetServerConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate project directory if provided
	if req.ProjectDir != "" {
		info, err := os.Stat(req.ProjectDir)
		if err != nil {
			http.Error(w, "Project directory does not exist", http.StatusBadRequest)
			return
		}
		if !info.IsDir() {
			http.Error(w, "Path is not a directory", http.StatusBadRequest)
			return
		}
	}

	// Save to server-project.json
	if err := config.SetServerProjectDir(req.ProjectDir); err != nil {
		http.Error(w, "Failed to save configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update the initial directory if explicit project dir is set
	if req.ProjectDir != "" {
		SetInitialDir(req.ProjectDir)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetEffectiveProjectDir returns the effective project directory
// Uses explicit config if set, otherwise falls back to auto-detected
func GetEffectiveProjectDir() string {
	// First check server-project.json
	if projectDir := config.GetServerProjectDir(); projectDir != "" {
		return projectDir
	}
	// Fall back to legacy config (for backward compatibility)
	if cfg := config.Get(); cfg != nil && cfg.Server.ProjectDir != "" {
		return cfg.Server.ProjectDir
	}
	return GetInitialDir()
}

// SetConfigFilePath sets the path to the configuration file
// Deprecated: Server project config is now stored in .ai-critic/server-project.json
// This function is kept for backward compatibility
func SetConfigFilePath(path string) {
	// No-op: config file path is no longer used for server settings
}
