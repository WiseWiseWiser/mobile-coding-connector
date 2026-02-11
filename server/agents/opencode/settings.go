// Package opencode provides an adapter for the OpenCode agent server,
// converting its native event format to standard ACP messages.
package opencode

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

// WebServerConfig holds the web server configuration.
type WebServerConfig struct {
	Enabled       bool   `json:"enabled"`
	Port          int    `json:"port"`
	ExposedDomain string `json:"exposed_domain,omitempty"`
	Password      string `json:"password,omitempty"`
}

// Settings holds the persisted opencode configuration.
type Settings struct {
	Model         string          `json:"model,omitempty"`
	DefaultDomain string          `json:"default_domain,omitempty"`
	WebServer     WebServerConfig `json:"web_server"`
}

var (
	settingsMu    sync.RWMutex
	settingsCache *Settings
)

// settingsPath returns the path to the settings file.
func settingsPath() string {
	return config.OpencodeFile
}

// LoadSettings loads the opencode settings from disk.
// Returns an empty settings struct if the file doesn't exist.
func LoadSettings() (*Settings, error) {
	settingsMu.RLock()
	if settingsCache != nil {
		defer settingsMu.RUnlock()
		return copySettings(settingsCache), nil
	}
	settingsMu.RUnlock()

	settingsMu.Lock()
	defer settingsMu.Unlock()

	// Double-check after acquiring write lock
	if settingsCache != nil {
		return copySettings(settingsCache), nil
	}

	s := &Settings{}
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		if os.IsNotExist(err) {
			// Set default values
			s.WebServer.Port = 4096
			settingsCache = s
			return copySettings(s), nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}

	// Set default port if not specified
	if s.WebServer.Port == 0 {
		s.WebServer.Port = 4096
	}

	settingsCache = s
	return copySettings(s), nil
}

// copySettings creates a copy of the settings.
func copySettings(s *Settings) *Settings {
	return &Settings{
		Model:         s.Model,
		DefaultDomain: s.DefaultDomain,
		WebServer: WebServerConfig{
			Enabled:       s.WebServer.Enabled,
			Port:          s.WebServer.Port,
			ExposedDomain: s.WebServer.ExposedDomain,
			Password:      s.WebServer.Password,
		},
	}
}

// SaveSettings saves the opencode settings to disk.
func SaveSettings(s *Settings) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return err
	}

	// Set default port if not specified
	if s.WebServer.Port == 0 {
		s.WebServer.Port = 4096
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(settingsPath(), data, 0644); err != nil {
		return err
	}

	// Update cache
	settingsCache = copySettings(s)
	return nil
}

// GetModel returns the saved model, or empty string if none is set.
func GetModel() string {
	s, err := LoadSettings()
	if err != nil {
		return ""
	}
	return s.Model
}

// SetModel saves the model to settings.
func SetModel(model string) error {
	s, err := LoadSettings()
	if err != nil {
		s = &Settings{WebServer: WebServerConfig{Port: 4096}}
	}
	s.Model = model
	return SaveSettings(s)
}

// GetDefaultDomain returns the saved default domain, or empty string if none is set.
func GetDefaultDomain() string {
	s, err := LoadSettings()
	if err != nil {
		return ""
	}
	return s.DefaultDomain
}

// SetDefaultDomain saves the default domain to settings.
func SetDefaultDomain(domain string) error {
	s, err := LoadSettings()
	if err != nil {
		s = &Settings{WebServer: WebServerConfig{Port: 4096}}
	}
	s.DefaultDomain = domain
	return SaveSettings(s)
}

// GetWebServerConfig returns the web server configuration.
func GetWebServerConfig() WebServerConfig {
	s, err := LoadSettings()
	if err != nil {
		return WebServerConfig{Port: 4096}
	}
	return s.WebServer
}

// SetWebServerConfig saves the web server configuration.
func SetWebServerConfig(cfg WebServerConfig) error {
	s, err := LoadSettings()
	if err != nil {
		s = &Settings{WebServer: cfg}
	}
	s.WebServer = cfg
	return SaveSettings(s)
}
