package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

var configFilePath string

// SetConfigFilePath sets the path to the configuration file
func SetConfigFilePath(path string) {
	configFilePath = path
}

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

	// Get explicit project dir from config
	var explicitProjectDir string
	if cfg := config.Get(); cfg != nil {
		explicitProjectDir = cfg.Server.ProjectDir
	}

	resp := ServerConfigResponse{
		ProjectDir:       explicitProjectDir,
		AutoDetectedDir:  autoDetectedDir,
		UsingExplicitDir: explicitProjectDir != "",
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

	// Get current config
	cfg := config.Get()
	if cfg == nil {
		// Create a new config if none exists
		cfg = &config.Config{}
		config.Set(cfg)
	}

	// Update the server config
	cfg.Server.ProjectDir = req.ProjectDir

	// Save to file if we have a config file path
	if configFilePath != "" {
		if err := saveConfigToFile(cfg, configFilePath); err != nil {
			http.Error(w, "Failed to save configuration", http.StatusInternalServerError)
			return
		}
	}

	// Update the initial directory if explicit project dir is set
	if req.ProjectDir != "" {
		SetInitialDir(req.ProjectDir)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// saveConfigToFile saves the configuration to the specified file
func saveConfigToFile(cfg *config.Config, path string) error {
	// Read existing file to preserve other fields
	var existingConfig config.Config
	data, err := os.ReadFile(path)
	if err == nil {
		// File exists, parse it
		if err := json.Unmarshal(data, &existingConfig); err == nil {
			// Preserve AI and PortForwarding configs
			if len(cfg.AI.Providers) == 0 && len(cfg.AI.Models) == 0 {
				cfg.AI = existingConfig.AI
			}
			if len(cfg.PortForwarding.Providers) == 0 {
				cfg.PortForwarding = existingConfig.PortForwarding
			}
		}
	}

	// Update server config
	existingConfig.Server = cfg.Server

	// Marshal and save
	data, err = json.MarshalIndent(existingConfig, "", "    ")
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return os.WriteFile(path, data, 0644)
}

// GetEffectiveProjectDir returns the effective project directory
// Uses explicit config if set, otherwise falls back to auto-detected
func GetEffectiveProjectDir() string {
	if cfg := config.Get(); cfg != nil && cfg.Server.ProjectDir != "" {
		return cfg.Server.ProjectDir
	}
	return GetInitialDir()
}
