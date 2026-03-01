package proxyconfig

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

var (
	configFileMu   sync.RWMutex
	configFilePath = config.DataDir + "/proxy-servers.json"
)

// SetConfigFile sets the path to the proxy config JSON file.
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

// ProxyServer represents a single proxy server configuration
type ProxyServer struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Protocol string   `json:"protocol,omitempty"` // http, https, socks5 (default: http)
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	Domains  []string `json:"domains"` // List of domains that should use this proxy
}

// ProxyConfig holds all proxy server configurations
type ProxyConfig struct {
	Enabled bool           `json:"enabled"` // Global enable/disable for proxy functionality
	Servers []*ProxyServer `json:"servers"`
}

// LoadConfig reads the proxy config from disk.
func LoadConfig() (*ProxyConfig, error) {
	data, err := os.ReadFile(getConfigFile())
	if err != nil {
		if os.IsNotExist(err) {
			return &ProxyConfig{
				Enabled: true,
				Servers: []*ProxyServer{},
			}, nil
		}
		return nil, err
	}
	var cfg ProxyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// Ensure servers slice is not nil
	if cfg.Servers == nil {
		cfg.Servers = []*ProxyServer{}
	}
	return &cfg, nil
}

// SaveConfig writes the proxy config to disk.
func SaveConfig(cfg *ProxyConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigFile(), data, 0644)
}

// RegisterAPI registers the proxy config API endpoints.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/proxy/servers", handleProxyServers)
	mux.HandleFunc("/api/proxy/config", handleProxyConfig)
	mux.HandleFunc("/api/proxy/test", handleProxyTest)
}

func handleProxyServers(w http.ResponseWriter, r *http.Request) {
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
		json.NewEncoder(w).Encode(cfg.Servers)
	case http.MethodPut:
		var servers []*ProxyServer
		if err := json.NewDecoder(r.Body).Decode(&servers); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}
		cfg, err := LoadConfig()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		cfg.Servers = servers
		if err := SaveConfig(cfg); err != nil {
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

func handleProxyTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProxyURL  string `json:"proxyUrl"`
		TargetURL string `json:"targetUrl"`
		Enabled   bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.TargetURL == "" {
		http.Error(w, "targetUrl is required", http.StatusBadRequest)
		return
	}

	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	sw.SendLog("Testing proxy connection...")
	sw.SendLog(fmt.Sprintf("Target: %s", req.TargetURL))

	args := []string{"-v", "-s", "-o", "/dev/null", "-w", "\\nHTTP Code: %{http_code}\\nTime Total: %{time_total}s\\n", "--max-time", "15"}
	if req.Enabled && req.ProxyURL != "" {
		sw.SendLog(fmt.Sprintf("Proxy: %s", req.ProxyURL))
		args = append(args, "-x", req.ProxyURL)
	} else {
		sw.SendLog("Proxy: disabled (direct connection)")
	}
	args = append(args, req.TargetURL)

	sw.SendLog(fmt.Sprintf("Running: curl %s", req.TargetURL))
	cmd := exec.Command("curl", args...)
	err := sw.StreamCmd(cmd)
	if err != nil {
		sw.SendError(fmt.Sprintf("curl failed: %v", err))
		sw.SendDone(map[string]string{"success": "false"})
		return
	}

	sw.SendLog("Connection test completed successfully.")
	sw.SendDone(map[string]string{"success": "true", "message": "Proxy connection test passed"})
}

func handleProxyConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := LoadConfig()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		// Return just the enabled flag
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"enabled": cfg.Enabled})
	case http.MethodPost:
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
			return
		}
		cfg, err := LoadConfig()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		cfg.Enabled = req.Enabled
		if err := SaveConfig(cfg); err != nil {
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
