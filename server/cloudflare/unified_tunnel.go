package cloudflare

// Unified Tunnel Manager
//
// This package provides a unified Cloudflare tunnel manager that maintains a SINGLE
// cloudflared process for all port forwardings and domain tunnels.
//
// File Locations:
//   - .ai-critic/cloudflare-tunnel-gen.yml  - Auto-generated cloudflared config (DO NOT EDIT)
//   - .ai-critic/cloudflare-extra-mapping.json - User-defined extra mappings
//   - .ai-critic/cloudflare-tunnel-gen.yml.log - Tunnel process logs
//
// Extra Mappings Format (cloudflare-extra-mapping.json):
//   {
//     "mappings": [
//       {"domain": "example.com", "local_url": "http://localhost:8080"},
//       {"domain": "api.example.com", "local_url": "http://localhost:3000"}
//     ]
//   }
//
// Precedence Rules:
//   1. Server-configured mappings (from portforward API, domain tunnels) take precedence
//   2. Extra mappings are used only if the domain is not already configured by the server
//   3. If a domain exists in both, the server configuration wins

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"gopkg.in/yaml.v3"
)

const (
	// CloudflareTunnelGenConfig is the auto-generated cloudflare tunnel config file
	CloudflareTunnelGenConfig = config.DataDir + "/cloudflare-tunnel-gen.yml"
	// CloudflareExtraMappingFile is the user-defined extra mappings file
	CloudflareExtraMappingFile = config.DataDir + "/cloudflare-extra-mapping.json"
)

// ExtraMapping represents a single extra mapping from the JSON file
type ExtraMapping struct {
	Domain   string `json:"domain"`
	LocalURL string `json:"local_url"`
}

// ExtraMappingsConfig is the structure of the extra mappings JSON file
type ExtraMappingsConfig struct {
	Mappings []ExtraMapping `json:"mappings"`
}

// IngressMapping represents a single ingress rule for port forwarding
type IngressMapping struct {
	ID       string // unique identifier for this mapping (e.g., "port-8080" or "domain-example.com")
	Hostname string
	Service  string
	Source   string // e.g., "portforward:8080" or "domain:example.com"
}

// UnifiedTunnelManager manages a single cloudflare tunnel process
// that handles all port forwardings and domain tunnels.
type UnifiedTunnelManager struct {
	mu         sync.RWMutex
	mappings   map[string]*IngressMapping // keyed by ID
	cmd        *exec.Cmd
	config     *config.CloudflareTunnelConfig
	configPath string
	running    bool
	paused     bool // when true, health checks are skipped
}

var (
	// singleton instance
	unifiedManager     *UnifiedTunnelManager
	unifiedManagerOnce sync.Once
)

// GetUnifiedTunnelManager returns the singleton unified tunnel manager instance
func GetUnifiedTunnelManager() *UnifiedTunnelManager {
	unifiedManagerOnce.Do(func() {
		unifiedManager = &UnifiedTunnelManager{
			mappings: make(map[string]*IngressMapping),
		}
	})
	return unifiedManager
}

// SetConfig configures the tunnel manager with the cloudflare tunnel configuration
// Once set, the tunnel config cannot be changed - this ensures we always use one unified tunnel
func (utm *UnifiedTunnelManager) SetConfig(cfg config.CloudflareTunnelConfig) {
	utm.mu.Lock()
	defer utm.mu.Unlock()

	fmt.Printf("[unified-tunnel] SetConfig called: TunnelName=%s, TunnelID=%s, CredentialsFile=%s\n", cfg.TunnelName, cfg.TunnelID, cfg.CredentialsFile)

	if utm.config == nil {
		// First time setting config - use the provided tunnel
		fmt.Printf("[unified-tunnel] SetConfig: setting tunnel config: TunnelName=%s, TunnelID=%s\n", cfg.TunnelName, cfg.TunnelID)
		utm.config = &cfg
	} else {
		// Config already set - ignore and keep existing
		fmt.Printf("[unified-tunnel] SetConfig: WARNING - ignoring new tunnel config, keeping existing: TunnelName=%s, TunnelID=%s\n",
			utm.config.TunnelName, utm.config.TunnelID)
	}
}

// GetConfig returns the current tunnel config
func (utm *UnifiedTunnelManager) GetConfig() *config.CloudflareTunnelConfig {
	utm.mu.RLock()
	defer utm.mu.RUnlock()
	return utm.config
}

// AddMapping adds a new ingress mapping and restarts the tunnel if needed
func (utm *UnifiedTunnelManager) AddMapping(mapping *IngressMapping) error {
	utm.mu.Lock()
	defer utm.mu.Unlock()

	fmt.Printf("[unified-tunnel] AddMapping: id=%s hostname=%s service=%s\n", mapping.ID, mapping.Hostname, mapping.Service)

	if utm.config == nil {
		return fmt.Errorf("tunnel manager not configured")
	}

	// Check if this mapping already exists with same values
	if existing, ok := utm.mappings[mapping.ID]; ok {
		if existing.Hostname == mapping.Hostname && existing.Service == mapping.Service {
			// No change needed
			fmt.Printf("[unified-tunnel] AddMapping: mapping unchanged, skipping\n")
			return nil
		}
	}

	// Add or update the mapping
	utm.mappings[mapping.ID] = mapping
	fmt.Printf("[unified-tunnel] AddMapping: mapping added/updated, calling rebuildAndRestartLocked\n")

	// Rebuild config and restart if needed
	return utm.rebuildAndRestartLocked()
}

// RemoveMapping removes an ingress mapping and restarts the tunnel if needed
func (utm *UnifiedTunnelManager) RemoveMapping(id string) error {
	utm.mu.Lock()
	defer utm.mu.Unlock()

	fmt.Printf("[unified-tunnel] RemoveMapping: id=%s\n", id)

	if _, ok := utm.mappings[id]; !ok {
		fmt.Printf("[unified-tunnel] RemoveMapping: mapping not found, skipping\n")
		return nil // already removed
	}

	delete(utm.mappings, id)
	fmt.Printf("[unified-tunnel] RemoveMapping: mapping removed, calling rebuildAndRestartLocked\n")

	// Rebuild config and restart if needed
	return utm.rebuildAndRestartLocked()
}

// ListMappings returns all current server-configured ingress mappings
func (utm *UnifiedTunnelManager) ListMappings() []*IngressMapping {
	utm.mu.RLock()
	defer utm.mu.RUnlock()

	result := make([]*IngressMapping, 0, len(utm.mappings))
	for _, m := range utm.mappings {
		result = append(result, m)
	}

	// Sort by hostname for consistent ordering (maps have random iteration order in Go)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Hostname < result[j].Hostname
	})

	return result
}

// ListAllMappings returns all effective mappings (server + extra), with server mappings taking precedence
func (utm *UnifiedTunnelManager) ListAllMappings() []*IngressMapping {
	utm.mu.RLock()
	defer utm.mu.RUnlock()

	// Combine server mappings and extra mappings
	hostnameToMapping := make(map[string]*IngressMapping)

	// Add server mappings first (take precedence)
	for _, m := range utm.mappings {
		hostnameToMapping[m.Hostname] = m
	}

	// Add extra mappings only if hostname not already in server mappings
	extraMappings := utm.loadExtraMappings()
	for _, em := range extraMappings {
		if _, exists := hostnameToMapping[em.Domain]; !exists {
			hostnameToMapping[em.Domain] = &IngressMapping{
				ID:       "extra-" + em.Domain,
				Hostname: em.Domain,
				Service:  em.LocalURL,
				Source:   "extra-mapping",
			}
		}
	}

	// Convert map to slice
	// IMPORTANT: Sort by hostname for deterministic output (Go maps have random iteration order)
	result := make([]*IngressMapping, 0, len(hostnameToMapping))
	for _, m := range hostnameToMapping {
		result = append(result, m)
	}

	// Sort by hostname for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Hostname < result[j].Hostname
	})

	return result
}

// GetConfigPath returns the path to the auto-generated tunnel config file
func (utm *UnifiedTunnelManager) GetConfigPath() string {
	return CloudflareTunnelGenConfig
}

// GetExtraMappingsPath returns the path to the extra mappings JSON file
func (utm *UnifiedTunnelManager) GetExtraMappingsPath() string {
	return CloudflareExtraMappingFile
}

// loadExtraMappings loads extra mappings from the JSON file
func (utm *UnifiedTunnelManager) loadExtraMappings() []ExtraMapping {
	data, err := os.ReadFile(CloudflareExtraMappingFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return nil
	}

	var cfg ExtraMappingsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	return cfg.Mappings
}

// GetLogPath returns the path to the tunnel log file
func (utm *UnifiedTunnelManager) GetLogPath() string {
	return CloudflareTunnelGenConfig + ".log"
}

// ensureDataDir ensures the .ai-critic directory exists
func (utm *UnifiedTunnelManager) ensureDataDir() error {
	return os.MkdirAll(config.DataDir, 0755)
}

// rebuildAndRestartLocked rebuilds the config file and restarts the tunnel if changed
// Must be called with utm.mu held
func (utm *UnifiedTunnelManager) rebuildAndRestartLocked() error {
	fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: starting...\n")

	// Build new config
	newConfig := utm.buildConfig()
	fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: built config, mappings count: %d\n", len(utm.mappings))

	// Log current mappings
	for id, m := range utm.mappings {
		fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: mapping %s -> %s (%s)\n", id, m.Hostname, m.Service)
	}

	// Get config file path
	cfgPath := utm.GetConfigPath()

	// Ensure data directory exists before checking/writing
	if err := utm.ensureDataDir(); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Check if config has changed
	changed := utm.hasConfigChanged(cfgPath, newConfig)
	fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: hasConfigChanged=%v\n", changed)
	if !changed {
		fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: config unchanged, skipping restart\n")
		return nil // no change, skip restart
	}

	fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: config changed, BEFORE STOP - running=%v\n", utm.running)

	// Pause health checks during restart
	utm.paused = true
	fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: health checks paused\n")

	// Stop existing process
	utm.stopProcessLocked()
	fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: process stopped, AFTER STOP - running=%v\n", utm.running)

	// Write new config
	if err := WriteCloudflaredConfig(cfgPath, newConfig); err != nil {
		utm.paused = false
		return fmt.Errorf("failed to write config: %v", err)
	}
	fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: config written to %s\n", cfgPath)

	utm.configPath = cfgPath

	// Start new process
	fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: starting new process...\n")
	if err := utm.startProcessLocked(); err != nil {
		utm.paused = false
		return fmt.Errorf("failed to start tunnel: %v", err)
	}
	fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: process started successfully, AFTER START - running=%v\n", utm.running)

	// Resume health checks after a delay to allow tunnel to stabilize
	go func() {
		time.Sleep(15 * time.Second)
		utm.mu.Lock()
		utm.paused = false
		fmt.Printf("[unified-tunnel] rebuildAndRestartLocked: health checks resumed\n")
		utm.mu.Unlock()
	}()

	return nil
}

// buildConfig builds the CloudflaredConfig from current mappings and extra mappings
// Server-configured mappings take precedence over extra mappings (same domain = server wins)
// Must be called with utm.mu held
func (utm *UnifiedTunnelManager) buildConfig() *CloudflaredConfig {
	if utm.config == nil {
		return nil
	}

	tunnelRef := utm.config.TunnelName
	if tunnelRef == "" {
		tunnelRef = utm.config.TunnelID
	}

	// Resolve tunnel ID and credentials
	tunnelID, credFile := utm.resolveTunnelCreds(tunnelRef)

	// Collect all mappings in a map keyed by hostname
	// Server mappings are added first, then extra mappings only if hostname not already present
	hostnameToRule := make(map[string]IngressRule)

	// Add server-configured mappings first (these take precedence)
	for _, m := range utm.mappings {
		hostnameToRule[m.Hostname] = IngressRule{
			Hostname: m.Hostname,
			Service:  m.Service,
		}
	}

	// Add extra mappings from JSON file (only if hostname not already in server mappings)
	extraMappings := utm.loadExtraMappings()
	for _, em := range extraMappings {
		if _, exists := hostnameToRule[em.Domain]; !exists {
			hostnameToRule[em.Domain] = IngressRule{
				Hostname: em.Domain,
				Service:  em.LocalURL,
			}
		}
	}

	// Convert map to slice
	// IMPORTANT: Maps in Go have random iteration order, so we MUST sort
	// the rules by hostname to ensure consistent YAML generation.
	// Without sorting, the config file would change on every rebuild,
	// causing unnecessary tunnel restarts.
	rules := make([]IngressRule, 0, len(hostnameToRule)+1)
	for _, rule := range hostnameToRule {
		rules = append(rules, rule)
	}

	// Sort rules by hostname for deterministic YAML output
	// This prevents unnecessary tunnel restarts due to config file changes
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Hostname < rules[j].Hostname
	})

	// Add catch-all rule
	rules = append(rules, IngressRule{Service: "http_status:404"})

	return &CloudflaredConfig{
		Tunnel:          tunnelID,
		CredentialsFile: credFile,
		Ingress:         rules,
	}
}

// resolveTunnelCreds resolves tunnel ID and credentials file
func (utm *UnifiedTunnelManager) resolveTunnelCreds(tunnelRef string) (string, string) {
	if utm.config.TunnelID != "" && utm.config.CredentialsFile != "" {
		if _, err := os.Stat(utm.config.CredentialsFile); err == nil {
			return utm.config.TunnelID, utm.config.CredentialsFile
		}
	}

	// Fall back to auto-discovery
	tunnelID, credFile, err := EnsureTunnelExists(tunnelRef)
	if err != nil {
		// Log error but return empty values - will fail on start
		return "", ""
	}

	return tunnelID, credFile
}

// hasConfigChanged checks if the new config differs from what's on disk
func (utm *UnifiedTunnelManager) hasConfigChanged(cfgPath string, newConfig *CloudflaredConfig) bool {
	if newConfig == nil {
		fmt.Printf("[unified-tunnel] hasConfigChanged: newConfig is nil, returning false\n")
		return false
	}

	// Check if file exists
	existingData, err := os.ReadFile(cfgPath)
	if err != nil {
		// File doesn't exist or can't be read - treat as changed
		fmt.Printf("[unified-tunnel] hasConfigChanged: config file not found or error reading: %v, treating as changed\n", err)
		return true
	}

	// Marshal new config
	newData, err := yaml.Marshal(newConfig)
	if err != nil {
		// Can't marshal - treat as changed
		fmt.Printf("[unified-tunnel] hasConfigChanged: error marshaling config: %v, treating as changed\n", err)
		return true
	}

	// Compare
	existingTrimmed := bytes.TrimSpace(existingData)
	newTrimmed := bytes.TrimSpace(newData)
	eq := bytes.Equal(existingTrimmed, newTrimmed)
	fmt.Printf("[unified-tunnel] hasConfigChanged: comparing lengths old=%d new=%d, equal=%v\n", len(existingTrimmed), len(newTrimmed), eq)
	if !eq {
		fmt.Printf("[unified-tunnel] hasConfigChanged: old config:\n%s\n", string(existingTrimmed))
		fmt.Printf("[unified-tunnel] hasConfigChanged: new config:\n%s\n", string(newTrimmed))
	}
	return !eq
}

// startProcessLocked starts the cloudflared tunnel process
// Must be called with utm.mu held
func (utm *UnifiedTunnelManager) startProcessLocked() error {
	fmt.Printf("[unified-tunnel] startProcessLocked: starting...\n")
	if utm.config == nil {
		return fmt.Errorf("tunnel manager not configured")
	}

	tunnelRef := utm.config.TunnelName
	if tunnelRef == "" {
		tunnelRef = utm.config.TunnelID
	}

	cfgPath := utm.GetConfigPath()
	logPath := utm.GetLogPath()
	fmt.Printf("[unified-tunnel] startProcessLocked: tunnelRef=%s cfgPath=%s logPath=%s\n", tunnelRef, cfgPath, logPath)

	// Ensure data directory exists
	if err := utm.ensureDataDir(); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Open log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logFile = nil
		fmt.Printf("[unified-tunnel] startProcessLocked: could not open log file: %v\n", err)
	}

	// Kill any orphaned cloudflared processes using this config
	fmt.Printf("[unified-tunnel] startProcessLocked: killing orphaned processes\n")
	utm.killOrphanedProcess(cfgPath)

	// Start cloudflared
	cmd := exec.Command("cloudflared", "tunnel", "--config", cfgPath, "run", tunnelRef)
	fmt.Printf("[unified-tunnel] startProcessLocked: executing: cloudflared tunnel --config %s run %s\n", cfgPath, tunnelRef)

	if logFile != nil {
		cmd.Stdout = io.MultiWriter(os.Stdout, logFile)
		cmd.Stderr = io.MultiWriter(os.Stderr, logFile)
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// Run in its own process group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	if err := cmd.Start(); err != nil {
		if logFile != nil {
			logFile.Close()
		}
		fmt.Printf("[unified-tunnel] startProcessLocked: failed to start: %v\n", err)
		return err
	}

	utm.cmd = cmd
	utm.running = true
	fmt.Printf("[unified-tunnel] startProcessLocked: process started with PID %d\n", cmd.Process.Pid)

	// Start goroutine to wait for process
	go func() {
		fmt.Printf("[unified-tunnel] startProcessLocked: waiting for process to exit...\n")
		cmd.Wait()
		fmt.Printf("[unified-tunnel] startProcessLocked: process exited\n")
		if logFile != nil {
			logFile.Close()
		}
		utm.mu.Lock()
		utm.running = false
		utm.mu.Unlock()
	}()

	return nil
}

// stopProcessLocked stops the running cloudflared process
// Must be called with utm.mu held
func (utm *UnifiedTunnelManager) stopProcessLocked() {
	fmt.Printf("[unified-tunnel] stopProcessLocked: starting... cmd=%+v\n", utm.cmd)
	if utm.cmd == nil || utm.cmd.Process == nil {
		fmt.Printf("[unified-tunnel] stopProcessLocked: no process to stop\n")
		return
	}

	// Get tunnel ID from config for explicit shutdown
	tunnelID := ""
	if utm.config != nil {
		tunnelID = utm.config.TunnelID
		if tunnelID == "" {
			tunnelID = utm.config.TunnelName
		}
	}
	pid := utm.cmd.Process.Pid

	// Try graceful shutdown first
	fmt.Printf("[unified-tunnel] stopProcessLocked: sending SIGTERM to PID %d\n", pid)
	utm.cmd.Process.Signal(syscall.SIGTERM)

	// Wait up to 5 seconds for graceful shutdown
	done := make(chan struct{})
	go func() {
		utm.cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Graceful shutdown completed
		fmt.Printf("[unified-tunnel] stopProcessLocked: process terminated gracefully\n")
	case <-time.After(5 * time.Second):
		// Force kill
		fmt.Printf("[unified-tunnel] stopProcessLocked: graceful shutdown timed out, sending SIGKILL\n")
		utm.cmd.Process.Kill()
		utm.cmd.Wait()
		fmt.Printf("[unified-tunnel] stopProcessLocked: process killed\n")
	}

	// Cleanup tunnel connections via cloudflared to ensure clean shutdown
	if tunnelID != "" {
		fmt.Printf("[unified-tunnel] stopProcessLocked: cleaning up tunnel %s connections\n", tunnelID)
		if out, err := exec.Command("cloudflared", "tunnel", "cleanup", tunnelID).CombinedOutput(); err != nil {
			fmt.Printf("[unified-tunnel] stopProcessLocked: tunnel cleanup output: %s, err: %v\n", string(out), err)
		} else {
			fmt.Printf("[unified-tunnel] stopProcessLocked: tunnel cleanup succeeded: %s\n", string(out))
		}
		// Also try to cleanup any lingering processes
		if out, err := exec.Command("pkill", "-f", fmt.Sprintf("cloudflared.*%s", tunnelID)).CombinedOutput(); err == nil {
			fmt.Printf("[unified-tunnel] stopProcessLocked: killed lingering processes: %s\n", string(out))
		}
	}

	utm.cmd = nil
	utm.running = false
	fmt.Printf("[unified-tunnel] stopProcessLocked: done\n")
}

// killOrphanedProcess kills any cloudflared processes using the given config
func (utm *UnifiedTunnelManager) killOrphanedProcess(cfgPath string) {
	out, err := exec.Command("pgrep", "-f", "cloudflared.*"+cfgPath).Output()
	if err != nil {
		return // no matching process
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var pid int
		if _, err := fmt.Sscanf(line, "%d", &pid); err == nil && pid > 0 {
			if p, err := os.FindProcess(pid); err == nil {
				p.Kill()
			}
		}
	}
	time.Sleep(500 * time.Millisecond)
}

// Stop stops the unified tunnel manager and kills the cloudflared process
func (utm *UnifiedTunnelManager) Stop() {
	utm.mu.Lock()
	defer utm.mu.Unlock()
	utm.stopProcessLocked()
}

// IsRunning returns whether the tunnel process is currently running
func (utm *UnifiedTunnelManager) IsRunning() bool {
	utm.mu.RLock()
	defer utm.mu.RUnlock()
	return utm.running
}

// GetTunnelStatus returns the current status of the unified tunnel
func (utm *UnifiedTunnelManager) GetTunnelStatus() map[string]interface{} {
	utm.mu.RLock()
	defer utm.mu.RUnlock()

	status := map[string]interface{}{
		"running":     utm.running,
		"mappings":    len(utm.mappings),
		"config_path": utm.configPath,
	}

	if utm.config != nil {
		status["tunnel_name"] = utm.config.TunnelName
		status["tunnel_id"] = utm.config.TunnelID
		status["base_domain"] = utm.config.BaseDomain
	}

	return status
}

// CreateDNSRoutes creates DNS routes for all mappings
func (utm *UnifiedTunnelManager) CreateDNSRoutes() error {
	utm.mu.RLock()
	defer utm.mu.RUnlock()

	if utm.config == nil {
		return fmt.Errorf("tunnel manager not configured")
	}

	tunnelRef := utm.config.TunnelName
	if tunnelRef == "" {
		tunnelRef = utm.config.TunnelID
	}

	var errs []string
	for _, m := range utm.mappings {
		if err := CreateDNSRoute(tunnelRef, m.Hostname); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", m.Hostname, err))
		}
	}

	// Also create DNS routes for extra mappings
	extraMappings := utm.loadExtraMappings()
	for _, em := range extraMappings {
		// Check if this domain is already in server mappings (if so, skip)
		existsInServerMappings := false
		for _, m := range utm.mappings {
			if m.Hostname == em.Domain {
				existsInServerMappings = true
				break
			}
		}
		if existsInServerMappings {
			continue
		}
		if err := CreateDNSRoute(tunnelRef, em.Domain); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", em.Domain, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to create some DNS routes: %s", strings.Join(errs, "; "))
	}

	return nil
}

// MappingHealthCallback is called when a mapping's health status changes
type MappingHealthCallback func(mappingID, hostname string, healthy bool, consecutiveFailures int)

// StartHealthChecks starts a goroutine that monitors all mappings and calls the callback
// when health status changes. It checks each mapping every 10 seconds.
// After 3 consecutive failures, the callback is called with healthy=false.
func (utm *UnifiedTunnelManager) StartHealthChecks(callback MappingHealthCallback) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		type healthState struct {
			consecutiveFailures int
			lastHealthy         bool
		}

		// Track health state for each mapping
		states := make(map[string]*healthState)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		// Wait a bit before first check to allow tunnel to be ready
		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				utm.mu.RLock()
				paused := utm.paused
				mappings := make([]*IngressMapping, 0, len(utm.mappings))
				for _, m := range utm.mappings {
					mappings = append(mappings, m)
				}
				utm.mu.RUnlock()

				if paused {
					fmt.Printf("[unified-tunnel] StartHealthChecks: health checks paused, skipping\n")
					continue
				}

				fmt.Printf("[unified-tunnel] StartHealthChecks: checking %d mappings\n", len(mappings))
				for _, m := range mappings {
					fmt.Printf("[unified-tunnel] StartHealthChecks: checking mapping id=%s hostname=%s\n", m.ID, m.Hostname)
					healthy := utm.checkMappingHealth(m.Hostname)

					state, exists := states[m.ID]
					if !exists {
						state = &healthState{lastHealthy: true}
						states[m.ID] = state
					}

					if healthy {
						if !state.lastHealthy {
							// Recovered
							state.consecutiveFailures = 0
							state.lastHealthy = true
							if callback != nil {
								callback(m.ID, m.Hostname, true, 0)
							}
						}
					} else {
						state.consecutiveFailures++
						state.lastHealthy = false
						if callback != nil {
							callback(m.ID, m.Hostname, false, state.consecutiveFailures)
						}
					}
				}
			}
		}
	}()

	return cancel
}

// checkMappingHealth checks if a mapping's hostname is reachable via HTTPS ping
// It checks root path and /ping, accepting any 2xx/3xx or 530 as "healthy"
func (utm *UnifiedTunnelManager) checkMappingHealth(hostname string) bool {
	fmt.Printf("[unified-tunnel] checkMappingHealth: checking health for hostname=%s\n", hostname)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	urls := []string{
		fmt.Sprintf("https://%s/", hostname),
		fmt.Sprintf("https://%s/ping", hostname),
	}

	for _, url := range urls {
		fmt.Printf("[unified-tunnel] checkMappingHealth: trying %s\n", url)
		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("[unified-tunnel] checkMappingHealth: %s failed: %v\n", url, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			fmt.Printf("[unified-tunnel] checkMappingHealth: %s returned status %d, healthy=true\n", url, resp.StatusCode)
			return true
		}
		fmt.Printf("[unified-tunnel] checkMappingHealth: %s returned status %d, unhealthy\n", url, resp.StatusCode)
	}

	fmt.Printf("[unified-tunnel] checkMappingHealth: all URLs failed for %s, marking unhealthy\n", hostname)
	return false
}

// RestartMapping triggers a single tunnel restart to refresh the connection
// The previous implementation did remove+add which caused double restarts - now we just do one restart
func (utm *UnifiedTunnelManager) RestartMapping(mappingID string) error {
	fmt.Printf("[unified-tunnel] RestartMapping: triggering restart for mappingID=%s\n", mappingID)

	utm.mu.Lock()
	_, exists := utm.mappings[mappingID]
	if !exists {
		utm.mu.Unlock()
		return fmt.Errorf("mapping %s not found", mappingID)
	}

	// Log current state before restart
	fmt.Printf("[unified-tunnel] RestartMapping: current state - running=%v, pid=%d\n", utm.running, func() int {
		if utm.cmd != nil && utm.cmd.Process != nil {
			return utm.cmd.Process.Pid
		}
		return -1
	}())

	utm.mu.Unlock()

	fmt.Printf("[unified-tunnel] RestartMapping: calling rebuildAndRestartLocked\n")
	err := utm.rebuildAndRestartLocked()

	// Log state after restart
	utm.mu.Lock()
	fmt.Printf("[unified-tunnel] RestartMapping: after restart - running=%v, pid=%d, err=%v\n", utm.running, func() int {
		if utm.cmd != nil && utm.cmd.Process != nil {
			return utm.cmd.Process.Pid
		}
		return -1
	}(), err)
	utm.mu.Unlock()

	// Run cloudflared tunnel info to check status
	fmt.Printf("[unified-tunnel] RestartMapping: checking tunnel status...\n")
	tunnelID := ""
	utm.mu.Lock()
	if utm.config != nil {
		tunnelID = utm.config.TunnelID
		if tunnelID == "" {
			tunnelID = utm.config.TunnelName
		}
	}
	utm.mu.Unlock()

	if tunnelID != "" {
		if out, err := exec.Command("cloudflared", "tunnel", "info", tunnelID).Output(); err == nil {
			fmt.Printf("[unified-tunnel] RestartMapping: tunnel info:\n%s\n", string(out))
		} else {
			fmt.Printf("[unified-tunnel] RestartMapping: failed to get tunnel info: %v\n", err)
		}
	}

	return err
}

// GetMapping returns a mapping by ID
func (utm *UnifiedTunnelManager) GetMapping(mappingID string) (*IngressMapping, bool) {
	utm.mu.RLock()
	defer utm.mu.RUnlock()

	m, exists := utm.mappings[mappingID]
	return m, exists
}

// globalHealthCheckCancel tracks the global health check cancel function
var globalHealthCheckCancel context.CancelFunc
var globalHealthCheckOnce sync.Once

// StartGlobalHealthChecks starts a global health check goroutine that monitors
// all mappings in the unified tunnel. It automatically restarts mappings after
// 3 consecutive failures.
func StartGlobalHealthChecks() {
	globalHealthCheckOnce.Do(func() {
		utm := GetUnifiedTunnelManager()
		fmt.Printf("[unified-tunnel] StartGlobalHealthChecks: setting up health check callback\n")

		globalHealthCheckCancel = utm.StartHealthChecks(func(mappingID, hostname string, healthy bool, consecutiveFailures int) {
			fmt.Printf("[unified-tunnel] healthCheckCallback: mappingID=%s hostname=%s healthy=%v failures=%d\n", mappingID, hostname, healthy, consecutiveFailures)
			if healthy {
				fmt.Printf("[unified-tunnel] Health check recovered for %s (%s)\n", hostname, mappingID)
			} else {
				fmt.Printf("[unified-tunnel] Health check failed for %s (%s): %d/3\n", hostname, mappingID, consecutiveFailures)
				if consecutiveFailures >= 3 {
					fmt.Printf("[unified-tunnel] Restarting mapping %s (%s) after 3 failures...\n", mappingID, hostname)
					if err := utm.RestartMapping(mappingID); err != nil {
						fmt.Printf("[unified-tunnel] Failed to restart mapping %s: %v\n", mappingID, err)
					} else {
						fmt.Printf("[unified-tunnel] Mapping %s restarted successfully\n", mappingID)
					}
				}
			}
		})
		fmt.Println("[unified-tunnel] Global health checks started")
	})
}

// StopGlobalHealthChecks stops the global health check goroutine
func StopGlobalHealthChecks() {
	if globalHealthCheckCancel != nil {
		globalHealthCheckCancel()
		globalHealthCheckCancel = nil
		fmt.Println("[unified-tunnel] Global health checks stopped")
	}
}

// LoadExtraMappingsFile loads all extra mappings from the JSON file
func (utm *UnifiedTunnelManager) LoadExtraMappingsFile() (*ExtraMappingsConfig, error) {
	data, err := os.ReadFile(CloudflareExtraMappingFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &ExtraMappingsConfig{Mappings: []ExtraMapping{}}, nil
		}
		return nil, err
	}

	var cfg ExtraMappingsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// SaveExtraMappingsFile saves extra mappings to the JSON file
func (utm *UnifiedTunnelManager) SaveExtraMappingsFile(cfg *ExtraMappingsConfig) error {
	if err := utm.ensureDataDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(CloudflareExtraMappingFile, append(data, '\n'), 0644)
}

// AddExtraMapping adds a mapping to the extra mappings file and triggers a tunnel restart if needed
func (utm *UnifiedTunnelManager) AddExtraMapping(domain, localURL string) error {
	utm.mu.Lock()
	defer utm.mu.Unlock()

	cfg, err := utm.LoadExtraMappingsFile()
	if err != nil {
		return err
	}

	// Check if domain already exists
	for i, m := range cfg.Mappings {
		if m.Domain == domain {
			// Update existing
			cfg.Mappings[i].LocalURL = localURL
			if err := utm.SaveExtraMappingsFile(cfg); err != nil {
				return err
			}
			return utm.rebuildAndRestartLocked()
		}
	}

	// Add new
	cfg.Mappings = append(cfg.Mappings, ExtraMapping{Domain: domain, LocalURL: localURL})
	if err := utm.SaveExtraMappingsFile(cfg); err != nil {
		return err
	}

	return utm.rebuildAndRestartLocked()
}

// RemoveExtraMapping removes a mapping from the extra mappings file and triggers a tunnel restart if needed
func (utm *UnifiedTunnelManager) RemoveExtraMapping(domain string) error {
	utm.mu.Lock()
	defer utm.mu.Unlock()

	cfg, err := utm.LoadExtraMappingsFile()
	if err != nil {
		return err
	}

	// Find and remove
	found := false
	newMappings := make([]ExtraMapping, 0, len(cfg.Mappings))
	for _, m := range cfg.Mappings {
		if m.Domain == domain {
			found = true
			continue
		}
		newMappings = append(newMappings, m)
	}

	if !found {
		return nil // not found, nothing to do
	}

	cfg.Mappings = newMappings
	if err := utm.SaveExtraMappingsFile(cfg); err != nil {
		return err
	}

	return utm.rebuildAndRestartLocked()
}
