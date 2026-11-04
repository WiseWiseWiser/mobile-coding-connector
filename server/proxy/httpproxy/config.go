// Package httpproxy implements the system HTTP forward proxy: an in-process
// listener that speaks plain HTTP proxy semantics (absolute-form GET/POST and
// CONNECT tunnels) guarded by mandatory Basic auth, so LAN and cable-adjacent
// clients can reach the intranet through this machine.
//
// It follows the wsproxy package shape: config persisted under config.DataDir,
// a singleton Manager with Start/Stop/Status, an BootAutoStart hook, and an
// api.go that registers JSON endpoints under /api/http-proxy.
package httpproxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xhd2015/ai-critic/server/config"
)

const (
	defaultListenPort  = 0 // 0 means auto-assign a random high port
	defaultUsername    = "proxy"
	defaultFailLimit   = 10  // failed auth attempts allowed per window per client
	defaultFailWindow  = 300 // seconds
	configFileName     = "http-proxy.json"
	logFileName        = "http-proxy.log"
	defaultPublishProv = "cloudflare_owned"
)

// Config is the persisted proxy configuration. Password is stored in plain
// text on purpose (0600 below): the detail page must be able to show the
// credential the user types into phone Wi-Fi settings, mirroring how wsproxy
// stores its VMess uuid.
type Config struct {
	ListenPort int `json:"listen_port"`
	// BindAddr defaults to 0.0.0.0: cable/LAN clients (e.g. an RSP with a
	// port mapping pointed at this proxy) must reach the listener.
	BindAddr string `json:"bind_addr,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	// Domain is the optional owned-domain label to publish this proxy under
	// via the cloudflare_owned port-forward provider (same stack user
	// services use). Empty means LAN/direct only.
	Domain    string `json:"domain,omitempty"`
	AutoStart bool   `json:"auto_start"`
	// AllowLoopbackConnect permits CONNECT tunnels to the proxy host's own
	// loopback. Off by default: an outside client must not use the proxy to
	// hop into local services.
	AllowLoopbackConnect bool `json:"allow_loopback_connect,omitempty"`
	// FailLimit / FailWindowSecs tune the per-client auth-failure lockout.
	FailLimit      int `json:"fail_limit,omitempty"`
	FailWindowSecs int `json:"fail_window_secs,omitempty"`
}

var _testConfigDir string

// SetConfigDir redirects config storage to dir; tests call it and must not
// race a running proxy.
func SetConfigDir(dir string) {
	_testConfigDir = dir
}

func configPath() string {
	if _testConfigDir != "" {
		return filepath.Join(_testConfigDir, configFileName)
	}
	return filepath.Join(config.DataDir, configFileName)
}

func LogPath() string {
	if _testConfigDir != "" {
		return filepath.Join(_testConfigDir, logFileName)
	}
	return filepath.Join(config.DataDir, logFileName)
}

// ConfigExists reports whether a proxy config has been saved. LoadConfig falls
// back to defaults without an error, so callers that must not act on an
// unconfigured proxy need this to tell the two cases apart.
func ConfigExists() bool {
	_, err := os.Stat(configPath())
	return err == nil
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if os.IsNotExist(err) {
		return defaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	cfg := defaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}

func defaultConfig() *Config {
	return &Config{
		ListenPort:     defaultListenPort,
		Username:       defaultUsername,
		FailLimit:      defaultFailLimit,
		FailWindowSecs: defaultFailWindow,
	}
}

// normalized applies defaults to optional fields after a load.
func (c *Config) normalized() {
	if c.BindAddr == "" {
		c.BindAddr = "0.0.0.0"
	}
	if c.Username == "" {
		c.Username = defaultUsername
	}
	if c.FailLimit <= 0 {
		c.FailLimit = defaultFailLimit
	}
	if c.FailWindowSecs <= 0 {
		c.FailWindowSecs = defaultFailWindow
	}
}
