package terminal

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
)

const defaultConfigFile = ".terminal-config.json"

var (
	configFileMu   sync.RWMutex
	configFilePath = defaultConfigFile
)

// SetConfigFile sets the path to the terminal config JSON file.
// Must be called before the server starts.
func SetConfigFile(path string) {
	configFileMu.Lock()
	defer configFileMu.Unlock()
	configFilePath = path
}

func getConfigFile() string {
	configFileMu.RLock()
	defer configFileMu.RUnlock()
	return configFilePath
}

// TerminalConfig is the persisted terminal configuration.
type TerminalConfig struct {
	ExtraPaths []string `json:"extra_paths"`
	Shell      string   `json:"shell,omitempty"`       // shell path or name (default: "bash")
	ShellFlags []string `json:"shell_flags,omitempty"` // shell flags (default: ["-i"])
}

// LoadConfig reads the terminal config from disk.
func LoadConfig() (*TerminalConfig, error) {
	data, err := os.ReadFile(getConfigFile())
	if err != nil {
		if os.IsNotExist(err) {
			return &TerminalConfig{}, nil
		}
		return nil, err
	}
	var cfg TerminalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig writes the terminal config to disk.
func SaveConfig(cfg *TerminalConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigFile(), data, 0644)
}

// handleConfig handles GET/POST for /api/terminal/config
func handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := LoadConfig()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	case http.MethodPost:
		var cfg TerminalConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}
		if err := SaveConfig(&cfg); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
