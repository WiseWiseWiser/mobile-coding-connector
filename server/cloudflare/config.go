package cloudflare

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

var (
	configFileMu   sync.RWMutex
	configFilePath = config.CloudflareFile
)

// SetConfigFile sets the path to the cloudflare config JSON file.
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

// CloudflareConfig holds cloudflare-specific configuration
// stored in .ai-critic/cloudflare.json.
type CloudflareConfig struct {
	OwnedDomains []string `json:"owned_domains"`
}

// LoadConfig reads the cloudflare config from disk.
func LoadConfig() (*CloudflareConfig, error) {
	data, err := os.ReadFile(getConfigFile())
	if os.IsNotExist(err) {
		return &CloudflareConfig{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg CloudflareConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig writes the cloudflare config to disk.
func SaveConfig(cfg *CloudflareConfig) error {
	data, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigFile(), append(data, '\n'), 0644)
}

// GetOwnedDomains returns the list of user-owned domains from cloudflare config.
func GetOwnedDomains() []string {
	cfg, err := LoadConfig()
	if err != nil {
		return nil
	}
	return cfg.OwnedDomains
}

func handleOwnedDomains(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := LoadConfig()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"owned_domains": cfg.OwnedDomains,
		})

	case http.MethodPost:
		var req struct {
			OwnedDomains []string `json:"owned_domains"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid request body")
			return
		}
		cfg, err := LoadConfig()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		cfg.OwnedDomains = req.OwnedDomains
		if err := SaveConfig(cfg); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":        "ok",
			"owned_domains": cfg.OwnedDomains,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
