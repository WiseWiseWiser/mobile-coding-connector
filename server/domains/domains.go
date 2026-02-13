package domains

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	cloudflareSettings "github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/domains/pick"
	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
)

var (
	domainsFileMu   sync.RWMutex
	domainsFilePath = config.DomainsFile

	serverPortMu sync.RWMutex
	serverPort   int

	// healthCheckCancel tracks cancel functions for domain health check goroutines
	healthCheckMu     sync.RWMutex
	healthCheckCancel = map[string]context.CancelFunc{}

	// healthCheckLogs stores log buffers for each domain's health check goroutine
	healthCheckLogsMu sync.RWMutex
	healthCheckLogs   = map[string]*healthCheckLogBuffer{}
)

const maxHealthCheckLogLines = 32

// healthCheckLogBuffer is a thread-safe circular buffer for health check logs
type healthCheckLogBuffer struct {
	mu    sync.Mutex
	lines []string
}

// newHealthCheckLogBuffer creates a new log buffer
func newHealthCheckLogBuffer() *healthCheckLogBuffer {
	return &healthCheckLogBuffer{
		lines: make([]string, 0, maxHealthCheckLogLines),
	}
}

// addLog adds a log line to the buffer
func (b *healthCheckLogBuffer) addLog(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.lines = append(b.lines, line)
	// Keep only last maxHealthCheckLogLines
	if len(b.lines) > maxHealthCheckLogLines {
		b.lines = b.lines[len(b.lines)-maxHealthCheckLogLines:]
	}
}

// getLogs returns a copy of all stored log lines
func (b *healthCheckLogBuffer) getLogs() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	result := make([]string, len(b.lines))
	copy(result, b.lines)
	return result
}

// getHealthCheckLogs retrieves the health check logs for a given domain
func getHealthCheckLogs(domain string) []string {
	healthCheckLogsMu.RLock()
	buffer, ok := healthCheckLogs[domain]
	healthCheckLogsMu.RUnlock()

	if !ok || buffer == nil {
		return []string{}
	}
	return buffer.getLogs()
}

// Provider constants
const (
	ProviderCloudflare = "cloudflare"
	ProviderNgrok      = "ngrok"
)

// SetDomainsFile sets the path to the domains JSON file.
// Must be called before the server starts.
func SetDomainsFile(path string) {
	domainsFileMu.Lock()
	defer domainsFileMu.Unlock()
	domainsFilePath = path
}

func getDomainsFile() string {
	domainsFileMu.RLock()
	defer domainsFileMu.RUnlock()
	return domainsFilePath
}

// SetServerPort stores the server port for tunnel target URL.
func SetServerPort(port int) {
	serverPortMu.Lock()
	defer serverPortMu.Unlock()
	serverPort = port
}

func getServerPort() int {
	serverPortMu.RLock()
	defer serverPortMu.RUnlock()
	return serverPort
}

// DomainEntry represents a configured domain with its tunnel provider
type DomainEntry struct {
	Domain   string `json:"domain"`
	Provider string `json:"provider"`
}

// DomainWithStatus extends DomainEntry with runtime tunnel status
type DomainWithStatus struct {
	DomainEntry
	Status    string `json:"status"`               // "stopped", "connecting", "active", "error"
	TunnelURL string `json:"tunnel_url,omitempty"` // actual tunnel URL when active
	Error     string `json:"error,omitempty"`
}

// DomainsConfig is the top-level JSON structure
type DomainsConfig struct {
	Domains    []DomainEntry `json:"domains"`
	TunnelName string        `json:"tunnel_name,omitempty"` // Cloudflare tunnel name, persisted
}

// DomainsWithStatusResponse includes status for each domain
type DomainsWithStatusResponse struct {
	Domains []DomainWithStatus `json:"domains"`
}

// LoadDomains reads the domains configuration from disk
func LoadDomains() (*DomainsConfig, error) {
	data, err := os.ReadFile(getDomainsFile())
	if err != nil {
		if os.IsNotExist(err) {
			return &DomainsConfig{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return &DomainsConfig{}, nil
	}
	var cfg DomainsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveDomains writes the domains configuration to disk
func SaveDomains(cfg *DomainsConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getDomainsFile(), append(data, '\n'), 0644)
}

// AutoStartTunnels starts Cloudflare tunnels for all configured domains.
// Should be called after SetServerPort and SetDomainsFile.
// Runs in the background and logs any errors to stdout.
func AutoStartTunnels() {
	cfg, err := LoadDomains()
	if err != nil {
		fmt.Printf("[domains] auto-start: failed to load domains config: %v\n", err)
		return
	}

	port := getServerPort()
	if port == 0 {
		return
	}

	tunnelName := cfg.TunnelName
	for _, d := range cfg.Domains {
		if d.Provider != ProviderCloudflare {
			continue
		}
		domain := d.Domain
		go func() {
			fmt.Printf("[domains] auto-starting tunnel for %s...\n", domain)
			logFn := func(msg string) {
				fmt.Printf("[domains] %s: %s\n", domain, msg)
			}
			_, err := cloudflareSettings.StartDomainTunnel(domain, port, tunnelName, logFn)
			if err != nil {
				fmt.Printf("[domains] auto-start failed for %s: %v\n", domain, err)
			} else {
				fmt.Printf("[domains] tunnel started for %s\n", domain)
				// Start health check goroutine for this domain
				startDomainHealthCheck(domain, port, tunnelName)
			}
		}()
	}
}

// RegisterAPI registers the domains endpoints
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/domains", handleDomains)
	mux.HandleFunc("/api/domains/cloudflare-status", handleCloudflareStatus)
	mux.HandleFunc("/api/domains/tunnel/start", handleTunnelStart)
	mux.HandleFunc("/api/domains/tunnel/stop", handleTunnelStop)
	mux.HandleFunc("/api/domains/tunnel-name", handleTunnelName)
	mux.HandleFunc("/api/domains/random-subdomain", handleRandomSubdomain)
	mux.HandleFunc("/api/domains/health-logs", handleHealthCheckLogs)
}

func handleDomains(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetDomains(w, r)
	case http.MethodPost:
		handleSaveDomains(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetDomains(w http.ResponseWriter, _ *http.Request) {
	cfg, err := LoadDomains()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := DomainsWithStatusResponse{
		Domains: make([]DomainWithStatus, 0, len(cfg.Domains)),
	}
	for _, d := range cfg.Domains {
		ds := DomainWithStatus{
			DomainEntry: d,
			Status:      "stopped",
		}
		if d.Provider == ProviderCloudflare {
			ts := cloudflareSettings.GetDomainTunnelStatus(d.Domain)
			ds.Status = ts.Status
			ds.TunnelURL = ts.TunnelURL
			ds.Error = ts.Error
		}
		resp.Domains = append(resp.Domains, ds)
	}

	writeJSON(w, resp)
}

func handleSaveDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var cfg DomainsConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := SaveDomains(&cfg); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Stop tunnels for removed domains
	activeDomains := make(map[string]bool, len(cfg.Domains))
	for _, d := range cfg.Domains {
		activeDomains[d.Domain] = true
	}
	// TODO: stop removed cloudflare domain tunnels

	writeJSON(w, map[string]string{"status": "ok"})
}

func handleCloudflareStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := cloudflareSettings.CheckStatus()
	writeJSON(w, map[string]any{
		"installed":     status.Installed,
		"authenticated": status.Authenticated,
		"auth_error":    status.Error,
	})
}

func handleTunnelStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Domain == "" {
		writeJSONError(w, http.StatusBadRequest, "domain is required")
		return
	}

	// Find the domain entry to get provider
	cfg, err := LoadDomains()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load domains: %v", err))
		return
	}

	var entry *DomainEntry
	for i := range cfg.Domains {
		if cfg.Domains[i].Domain == req.Domain {
			entry = &cfg.Domains[i]
			break
		}
	}
	if entry == nil {
		writeJSONError(w, http.StatusNotFound, "domain not found in config")
		return
	}

	if entry.Provider != ProviderCloudflare {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("provider %q does not support tunnels yet", entry.Provider))
		return
	}

	port := getServerPort()
	if port == 0 {
		writeJSONError(w, http.StatusInternalServerError, "server port not configured")
		return
	}

	// Stream tunnel start logs via SSE
	sw := sse.NewWriter(w)
	if sw == nil {
		writeJSONError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	tunnelName := cfg.TunnelName

	logFn := func(message string) {
		sw.SendLog(message)
	}

	status, err := cloudflareSettings.StartDomainTunnel(req.Domain, port, tunnelName, logFn)
	if err != nil {
		sw.SendError(fmt.Sprintf("Failed to start tunnel: %v", err))
		return
	}

	// Start health check for manually started tunnels too
	startDomainHealthCheck(req.Domain, port, tunnelName)

	sw.SendDone(map[string]string{
		"message":    "Tunnel started successfully",
		"status":     status.Status,
		"tunnel_url": status.TunnelURL,
	})
}

func handleTunnelStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Domain == "" {
		writeJSONError(w, http.StatusBadRequest, "domain is required")
		return
	}

	cfg, err := LoadDomains()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("failed to load domains: %v", err))
		return
	}

	// Stop health check for this domain
	stopDomainHealthCheck(req.Domain)

	if err := cloudflareSettings.StopDomainTunnel(req.Domain, cfg.TunnelName); err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, map[string]string{"status": "stopped"})
}

func handleTunnelName(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := LoadDomains()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]string{"tunnel_name": cfg.TunnelName})

	case http.MethodPost:
		var req struct {
			TunnelName string `json:"tunnel_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		cfg, err := LoadDomains()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		cfg.TunnelName = req.TunnelName
		if err := SaveDomains(cfg); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]string{"status": "ok", "tunnel_name": cfg.TunnelName})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleRandomSubdomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// If a current domain is provided, preserve its base domain suffix
	current := r.URL.Query().Get("current")
	subdomain := pick.RandomSubdomain()

	if current != "" {
		baseDomain := cloudflareSettings.ParseBaseDomain(current)
		if baseDomain != "" && baseDomain != current {
			// current has a base domain suffix, preserve it
			writeJSON(w, map[string]string{"domain": subdomain + "." + baseDomain})
			return
		}
		// current is itself a base domain (e.g. "example.com"), still append it
		if strings.Contains(current, ".") {
			writeJSON(w, map[string]string{"domain": subdomain + "." + current})
			return
		}
	}

	writeJSON(w, map[string]string{"domain": subdomain})
}

func handleHealthCheckLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeJSONError(w, http.StatusBadRequest, "domain parameter is required")
		return
	}

	logs := getHealthCheckLogs(domain)
	writeJSON(w, logs)
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// startDomainHealthCheck starts a health check goroutine for the given domain.
// It pings the domain's /ping endpoint every 10 seconds. If 3 consecutive pings
// fail, it stops the tunnel and restarts it.
func startDomainHealthCheck(domain string, port int, tunnelName string) {
	// Cancel any existing health check for this domain
	stopDomainHealthCheck(domain)

	ctx, cancel := context.WithCancel(context.Background())
	healthCheckMu.Lock()
	healthCheckCancel[domain] = cancel
	healthCheckMu.Unlock()

	// Create or get log buffer for this domain
	healthCheckLogsMu.Lock()
	logBuffer := newHealthCheckLogBuffer()
	healthCheckLogs[domain] = logBuffer
	healthCheckLogsMu.Unlock()

	go func() {
		defer func() {
			healthCheckMu.Lock()
			delete(healthCheckCancel, domain)
			healthCheckMu.Unlock()
		}()

		logBuffer.addLog(fmt.Sprintf("Health check started for %s", domain))

		consecutiveFailures := 0
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		// Wait a bit before first check to allow tunnel to be ready
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			logBuffer.addLog("Health check stopped")
			return
		}

		for {
			select {
			case <-ctx.Done():
				logBuffer.addLog("Health check stopped")
				return
			case <-ticker.C:
				if !checkDomainPing(domain) {
					consecutiveFailures++
					logMsg := fmt.Sprintf("Health check failed (%d/3)", consecutiveFailures)
					logBuffer.addLog(logMsg)
					fmt.Printf("[domains] %s: %s\n", domain, logMsg)
					if consecutiveFailures >= 3 {
						logBuffer.addLog("Health check failed 3 times, restarting mapping...")
						fmt.Printf("[domains] health check failed 3 times for %s, restarting mapping...\n", domain)

						// Use unified tunnel manager to restart the mapping
						utm := cloudflareSettings.GetUnifiedTunnelManager()
						mappingID := fmt.Sprintf("domain-%s", domain)
						if err := utm.RestartMapping(mappingID); err != nil {
							errMsg := fmt.Sprintf("Failed to restart mapping: %v", err)
							logBuffer.addLog(errMsg)
							fmt.Printf("[domains] %s: %s\n", domain, errMsg)
						} else {
							successMsg := "Mapping restarted successfully via unified tunnel"
							logBuffer.addLog(successMsg)
							fmt.Printf("[domains] %s: %s\n", domain, successMsg)
							// Reset failure counter after successful restart
							consecutiveFailures = 0
						}
					}
				} else {
					if consecutiveFailures > 0 {
						logBuffer.addLog("Health check recovered")
						fmt.Printf("[domains] health check recovered for %s\n", domain)
					}
					consecutiveFailures = 0
				}
			}
		}
	}()
}

// stopDomainHealthCheck stops the health check goroutine for the given domain.
func stopDomainHealthCheck(domain string) {
	healthCheckMu.Lock()
	cancel, ok := healthCheckCancel[domain]
	if ok {
		delete(healthCheckCancel, domain)
	}
	healthCheckMu.Unlock()
	if ok {
		cancel()
	}

	// Clean up log buffer
	healthCheckLogsMu.Lock()
	delete(healthCheckLogs, domain)
	healthCheckLogsMu.Unlock()
}

// checkDomainPing checks if the domain's /ping endpoint is reachable.
// Returns true if ping succeeds, false otherwise.
func checkDomainPing(domain string) bool {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	urls := []string{
		fmt.Sprintf("https://%s/", domain),
		fmt.Sprintf("https://%s/ping", domain),
	}

	for _, url := range urls {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			return true
		}
	}

	return false
}

// DomainTunnelInfo represents information about an active domain tunnel
type DomainTunnelInfo struct {
	Domain    string
	Provider  string
	Status    string
	TunnelURL string
	Error     string
}

// GetActiveDomainTunnels returns all configured domains with their tunnel status.
// This is used to include bootstrap domain tunnels in the port forwards list.
func GetActiveDomainTunnels() []DomainTunnelInfo {
	cfg, err := LoadDomains()
	if err != nil {
		return nil
	}

	var result []DomainTunnelInfo
	for _, d := range cfg.Domains {
		if d.Provider != ProviderCloudflare {
			continue
		}
		status := cloudflareSettings.GetDomainTunnelStatus(d.Domain)
		result = append(result, DomainTunnelInfo{
			Domain:    d.Domain,
			Provider:  d.Provider,
			Status:    status.Status,
			TunnelURL: status.TunnelURL,
			Error:     status.Error,
		})
	}
	return result
}

// GetServerPort returns the configured server port for domain tunnels.
// Returns 0 if not set.
func GetServerPort() int {
	return getServerPort()
}
