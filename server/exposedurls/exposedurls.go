package exposedurls

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	cf "github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/jsonfile"
)

// ExposedURL represents a single exposed URL configuration
type ExposedURL struct {
	ID             string `json:"id"`
	ExternalDomain string `json:"external_domain"`
	InternalURL    string `json:"internal_url"`
	CreatedAt      string `json:"created_at"`
}

// ExposedURLWithStatus extends ExposedURL with runtime status
type ExposedURLWithStatus struct {
	ExposedURL
	Status    string `json:"status"` // "stopped", "connecting", "active", "error"
	TunnelURL string `json:"tunnel_url,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Config represents the exposed URLs configuration
type Config struct {
	URLs []ExposedURL `json:"urls"`
}

// Manager handles exposed URLs configuration and runtime state
type Manager struct {
	jsonFile *jsonfile.JSONFile[Config]
	mu       sync.RWMutex
	urls     map[string]*ExposedURLWithStatus
	running  map[string]*runningTunnel
}

type runningTunnel struct {
	stop   func()
	logs   []string
	status string
}

var (
	defaultManager *Manager
	initOnce       sync.Once
)

// GetManager returns the singleton manager instance
func GetManager() *Manager {
	initOnce.Do(func() {
		configDir := getConfigDir()
		defaultManager = NewManager(filepath.Join(configDir, "exposed-urls.json"))
	})
	return defaultManager
}

// NewManager creates a new exposed URLs manager
func NewManager(configPath string) *Manager {
	jf := jsonfile.New[Config](configPath)
	if err := jf.Load(); err != nil {
		fmt.Printf("Warning: failed to load exposed URLs config: %v\n", err)
	}

	m := &Manager{
		jsonFile: jf,
		urls:     make(map[string]*ExposedURLWithStatus),
		running:  make(map[string]*runningTunnel),
	}

	// Load existing config into memory
	cfg, _ := jf.Get()
	for _, url := range cfg.URLs {
		m.urls[url.ID] = &ExposedURLWithStatus{
			ExposedURL: url,
			Status:     "stopped",
		}
	}

	return m
}

func getConfigDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".ai-critic")
	}
	return ".ai-critic"
}

// List returns all exposed URLs with their current status
func (m *Manager) List() []ExposedURLWithStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]ExposedURLWithStatus, 0, len(m.urls))
	for _, url := range m.urls {
		// Update status from running tunnels
		if rt, ok := m.running[url.ID]; ok {
			url.Status = rt.status
		}
		result = append(result, *url)
	}

	return result
}

// Get returns a single exposed URL by ID
func (m *Manager) Get(id string) (*ExposedURLWithStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	url, ok := m.urls[id]
	if !ok {
		return nil, fmt.Errorf("exposed URL not found: %s", id)
	}

	if rt, ok := m.running[id]; ok {
		url.Status = rt.status
	}

	return url, nil
}

// Add creates a new exposed URL
func (m *Manager) Add(externalDomain, internalURL string) (*ExposedURLWithStatus, error) {
	id := generateID()
	now := time.Now().UTC().Format(time.RFC3339)
	url := ExposedURL{
		ID:             id,
		ExternalDomain: externalDomain,
		InternalURL:    internalURL,
		CreatedAt:      now,
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.urls[id] = &ExposedURLWithStatus{
		ExposedURL: url,
		Status:     "stopped",
	}

	// Update config file
	err := m.jsonFile.Update(func(cfg *Config) error {
		cfg.URLs = append(cfg.URLs, url)
		return nil
	})
	if err != nil {
		delete(m.urls, id)
		return nil, err
	}

	return m.urls[id], nil
}

// Update modifies an existing exposed URL
func (m *Manager) Update(id, externalDomain, internalURL string) (*ExposedURLWithStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	url, ok := m.urls[id]
	if !ok {
		return nil, fmt.Errorf("exposed URL not found: %s", id)
	}

	url.ExternalDomain = externalDomain
	url.InternalURL = internalURL

	// Update config file
	err := m.jsonFile.Update(func(cfg *Config) error {
		for i := range cfg.URLs {
			if cfg.URLs[i].ID == id {
				cfg.URLs[i].ExternalDomain = externalDomain
				cfg.URLs[i].InternalURL = internalURL
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return url, nil
}

// Remove deletes an exposed URL and stops any running tunnel
func (m *Manager) Remove(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.urls[id]; !ok {
		return fmt.Errorf("exposed URL not found: %s", id)
	}

	// Stop running tunnel if any
	if rt, ok := m.running[id]; ok {
		rt.stop()
		delete(m.running, id)
	}

	delete(m.urls, id)

	// Update config file
	return m.jsonFile.Update(func(cfg *Config) error {
		newURLs := make([]ExposedURL, 0)
		for _, u := range cfg.URLs {
			if u.ID != id {
				newURLs = append(newURLs, u)
			}
		}
		cfg.URLs = newURLs
		return nil
	})
}

// StartTunnel starts a Cloudflare tunnel for the given exposed URL using the unified tunnel manager
func (m *Manager) StartTunnel(id string) error {
	m.mu.Lock()

	url, ok := m.urls[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("exposed URL not found: %s", id)
	}

	// Check if already running
	if rt, ok := m.running[id]; ok && rt.status == "active" {
		m.mu.Unlock()
		return fmt.Errorf("tunnel already running")
	}

	// Update status to connecting
	url.Status = "connecting"
	m.running[id] = &runningTunnel{
		status: "connecting",
	}

	// Capture values for goroutine
	tunnelID := id
	externalDomain := url.ExternalDomain
	internalURL := url.InternalURL
	m.mu.Unlock()

	// Ensure unified tunnel is configured before adding mapping
	utm := cf.GetUnifiedTunnelManager()
	logFn := func(msg string) {
		fmt.Printf("[exposed-urls] %s\n", msg)
	}
	_, _, _, err := cf.EnsureUnifiedTunnelConfigured("", logFn)
	if err != nil {
		m.mu.Lock()
		if rt, ok := m.running[tunnelID]; ok {
			rt.status = "error"
		}
		if url, ok := m.urls[tunnelID]; ok {
			url.Status = "error"
			url.Error = fmt.Sprintf("failed to configure unified tunnel: %v", err)
		}
		m.mu.Unlock()
		return fmt.Errorf("failed to configure unified tunnel: %v", err)
	}

	// Add mapping to unified tunnel manager
	mappingID := fmt.Sprintf("exposed-%s", tunnelID)
	mapping := &cf.IngressMapping{
		ID:       mappingID,
		Hostname: externalDomain,
		Service:  internalURL,
		Source:   fmt.Sprintf("exposed-url:%s", externalDomain),
	}

	if err := utm.AddMapping(mapping); err != nil {
		m.mu.Lock()
		if rt, ok := m.running[tunnelID]; ok {
			rt.status = "error"
		}
		if url, ok := m.urls[tunnelID]; ok {
			url.Status = "error"
			url.Error = fmt.Sprintf("failed to add mapping: %v", err)
		}
		m.mu.Unlock()
		return fmt.Errorf("failed to add mapping to unified tunnel: %v", err)
	}

	// Update status to active after a brief delay to allow tunnel to establish
	go func() {
		time.Sleep(3 * time.Second)
		m.mu.Lock()
		defer m.mu.Unlock()
		if rt, ok := m.running[tunnelID]; ok {
			rt.status = "active"
			if url, ok := m.urls[tunnelID]; ok {
				url.Status = "active"
				url.TunnelURL = fmt.Sprintf("https://%s", externalDomain)
			}
		}
	}()

	return nil
}

// StopTunnel stops the Cloudflare tunnel for the given exposed URL
func (m *Manager) StopTunnel(id string) error {
	m.mu.Lock()

	url, ok := m.urls[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("exposed URL not found: %s", id)
	}

	// Remove mapping from unified tunnel
	utm := cf.GetUnifiedTunnelManager()
	mappingID := fmt.Sprintf("exposed-%s", id)
	if err := utm.RemoveMapping(mappingID); err != nil {
		fmt.Printf("[exposed-urls] warning: failed to remove mapping: %v\n", err)
	}

	if rt, ok := m.running[id]; ok {
		if rt.stop != nil {
			rt.stop()
		}
		delete(m.running, id)
	}

	url.Status = "stopped"
	url.TunnelURL = ""
	url.Error = ""

	m.mu.Unlock()
	return nil
}

// InitExposedURLTunnels adds all exposed URLs to the unified tunnel on server startup.
func InitExposedURLTunnels() {
	m := GetManager()
	m.mu.RLock()
	urls := make([]ExposedURL, 0, len(m.urls))
	for _, url := range m.urls {
		urls = append(urls, url.ExposedURL)
	}
	m.mu.RUnlock()

	if len(urls) == 0 {
		fmt.Printf("[exposed-urls] No exposed URLs configured, skipping tunnel initialization\n")
		return
	}

	// Ensure unified tunnel is configured
	utm := cf.GetUnifiedTunnelManager()
	logFn := func(msg string) {
		fmt.Printf("[exposed-urls] %s\n", msg)
	}
	tunnelRef, _, _, err := cf.EnsureUnifiedTunnelConfigured("", logFn)
	if err != nil {
		fmt.Printf("[exposed-urls] Failed to configure unified tunnel: %v\n", err)
		return
	}
	fmt.Printf("[exposed-urls] Using unified tunnel: %s\n", tunnelRef)

	// Add each exposed URL as a mapping
	for _, url := range urls {
		mappingID := fmt.Sprintf("exposed-%s", url.ID)
		mapping := &cf.IngressMapping{
			ID:       mappingID,
			Hostname: url.ExternalDomain,
			Service:  url.InternalURL,
			Source:   fmt.Sprintf("exposed-url:%s", url.ExternalDomain),
		}

		if err := utm.AddMapping(mapping); err != nil {
			fmt.Printf("[exposed-urls] Failed to add mapping for %s: %v\n", url.ExternalDomain, err)
		} else {
			fmt.Printf("[exposed-urls] Added exposed URL mapping: %s -> %s\n", url.ExternalDomain, url.InternalURL)
		}
	}
}

// CheckCloudflareStatus checks if Cloudflare is properly configured
func (m *Manager) CheckCloudflareStatus() (installed, authenticated bool, err error) {
	// Check if cloudflared is installed
	if _, err := os.Stat("/usr/bin/cloudflared"); err == nil {
		installed = true
	} else if _, err := os.Stat("/usr/local/bin/cloudflared"); err == nil {
		installed = true
	} else {
		// Try to find in PATH
		_, err := os.Stat(filepath.Join(getConfigDir(), ".cloudflared", "cert.pem"))
		installed = err == nil
	}

	if !installed {
		return false, false, nil
	}

	// Check if authenticated
	cloudflaredDir := filepath.Join(getConfigDir(), ".cloudflared")
	if _, err := os.Stat(filepath.Join(cloudflaredDir, "cert.pem")); err == nil {
		authenticated = true
	}

	return installed, authenticated, nil
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
