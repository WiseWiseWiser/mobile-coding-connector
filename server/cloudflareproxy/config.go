package cloudflareproxy

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DirName        = "cloudflare-proxy"
	ConfigFileName = "config.json"
	MappingsFile   = "mappings.json"
	LogFileName    = "cloudflare-proxy.log"
	SocketFileName = "control.sock"

	DefaultPort     = 23790
	DefaultPoolSize = 8
	maxBodyBytes    = 8 << 20
)

// Config is ~/.ai-critic/cloudflare-proxy/config.json.
type Config struct {
	Token          string `json:"token"`
	Domain         string `json:"domain"`
	GeneratedToken string `json:"generatedToken,omitempty"`
	AutoStart      bool   `json:"autoStart,omitempty"`
	Port           int    `json:"port,omitempty"`
}

func (c Config) port() int {
	if c.Port > 0 {
		return c.Port
	}
	return DefaultPort
}

func (c Config) authToken() string {
	return c.AuthToken()
}

// AuthToken is the Bearer secret: configured token, else generatedToken.
func (c Config) AuthToken() string {
	if t := trim(c.Token); t != "" {
		return t
	}
	return trim(c.GeneratedToken)
}

func (c Config) authSource() string {
	if trim(c.Token) != "" {
		return "token"
	}
	if trim(c.GeneratedToken) != "" {
		return "generatedToken"
	}
	return "none"
}

// ClientFile is the work.dev/client slice of ~/.ai-critic/cloudflare.json.
type ClientFile struct {
	ProxyURL string `json:"proxy_url"`
	Token    string `json:"token"`
}

// BaseDir returns ~/.ai-critic/cloudflare-proxy.
func BaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ai-critic", DirName), nil
}

func ConfigPath(base string) string { return filepath.Join(base, ConfigFileName) }
func MappingsPath(base string) string {
	return filepath.Join(base, MappingsFile)
}
func LogPath(base string) string    { return filepath.Join(base, LogFileName) }
func SocketPath(base string) string { return filepath.Join(base, SocketFileName) }

// LoadConfig reads config.json; missing file yields a zero config.
func LoadConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// SaveConfig atomically writes config.json (0600).
func SaveConfig(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// LoadClientFile reads proxy_url/token from cloudflare.json without touching owned_domains.
func LoadClientFile(path string) (ClientFile, error) {
	var cf ClientFile
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cf, nil
		}
		return cf, err
	}
	if err := json.Unmarshal(data, &cf); err != nil {
		return cf, err
	}
	return cf, nil
}

func trim(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
