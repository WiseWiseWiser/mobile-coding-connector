package exposed_opencode

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

type TargetPreference string

const (
	TargetPreferenceDomain    TargetPreference = "domain"
	TargetPreferenceLocalhost TargetPreference = "localhost"
)

func NormalizeTargetPreference(pref TargetPreference) TargetPreference {
	if pref == TargetPreferenceLocalhost {
		return pref
	}
	return TargetPreferenceDomain
}

// WebServerConfig holds the web server configuration.
// These are user-configured preferences, NOT runtime state.
// The Enabled field indicates user's preference to auto-start on boot (not current running status).
// The Port field is the user-configured port preference, not the actual runtime port.
type WebServerConfig struct {
	Enabled          bool             `json:"enabled"`                     // User preference: auto-start on boot
	Port             int              `json:"port"`                        // User preference: desired port (default 4096)
	TargetPreference TargetPreference `json:"target_preference,omitempty"` // Preferred web target: domain or localhost
	ExposedDomain    string           `json:"exposed_domain,omitempty"`
	Password         string           `json:"password,omitempty"`
	AuthProxyEnabled bool             `json:"auth_proxy_enabled"`
}

// Settings holds the persisted opencode configuration.
//
// This struct stores user-configured settings only. Runtime state such as whether
// the web server is currently running should NOT be stored here - it will be
// overwritten when the settings are saved. Use in-memory tracking for runtime state.
type Settings struct {
	Model         string          `json:"model,omitempty"`
	DefaultDomain string          `json:"default_domain,omitempty"`
	BinaryPath    string          `json:"binary_path,omitempty"`
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
			s.WebServer.TargetPreference = TargetPreferenceDomain
			settingsCache = s
			return copySettings(s), nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}

	// Set defaults if not specified
	if s.WebServer.Port == 0 {
		s.WebServer.Port = 4096
	}
	s.WebServer.TargetPreference = NormalizeTargetPreference(s.WebServer.TargetPreference)

	settingsCache = s
	return copySettings(s), nil
}

// copySettings creates a copy of the settings.
func copySettings(s *Settings) *Settings {
	return &Settings{
		Model:         s.Model,
		DefaultDomain: s.DefaultDomain,
		BinaryPath:    s.BinaryPath,
		WebServer: WebServerConfig{
			Enabled:          s.WebServer.Enabled,
			Port:             s.WebServer.Port,
			TargetPreference: s.WebServer.TargetPreference,
			ExposedDomain:    s.WebServer.ExposedDomain,
			Password:         s.WebServer.Password,
			AuthProxyEnabled: s.WebServer.AuthProxyEnabled,
		},
	}
}

// SaveSettings saves the opencode settings to disk.
//
// IMPORTANT: This function should only be used to save settings explicitly configured by the user.
// DO NOT use this function to persist runtime state changes such as:
//   - Web server running status (Enabled)
//   - Dynamically allocated ports (Port)
//   - Any other runtime-computed values
//
// The settings file is user configuration, not runtime state. Runtime state should be
// tracked in memory or in separate runtime state files. Persisting runtime state to this
// file will overwrite user configuration and cause bugs like port being overwritten
// with auto-allocated ports, or Enabled being set incorrectly.
//
// For runtime status, use in-memory tracking or a separate runtime state mechanism.
func SaveSettings(s *Settings) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return err
	}

	// Set defaults if not specified
	if s.WebServer.Port == 0 {
		s.WebServer.Port = 4096
	}
	s.WebServer.TargetPreference = NormalizeTargetPreference(s.WebServer.TargetPreference)

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
