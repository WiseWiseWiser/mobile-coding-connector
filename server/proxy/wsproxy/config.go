package wsproxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xhd2015/ai-critic/server/config"
)

const (
	defaultListenPort    = 0 // 0 means auto-assign random high port
	defaultWSPath        = "/ws"
	defaultSubdomain     = "ws"
	configFileName       = "ws-proxy.json"
	xrayDirName          = "xray"
	xrayBinaryName       = "xray"
	xrayConfigFileName   = "config.json"
	xrayDownloadURLAmd64 = "https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip"
	portRangeLow         = 30000
	portRangeHigh        = 50000
)

type Config struct {
	UpstreamProxy string `json:"upstream_proxy"`
	ListenPort    int    `json:"listen_port"`
	WSPath        string `json:"ws_path"`
	UUID          string `json:"uuid"`
	Subdomain     string `json:"subdomain"`
	InstanceID    string `json:"instance_id"`
	AutoStart     bool   `json:"auto_start"`
}

var _testConfigDir string

func configPath() string {
	if _testConfigDir != "" {
		return filepath.Join(_testConfigDir, configFileName)
	}
	return filepath.Join(config.DataDir, configFileName)
}

func xrayDir() string {
	return filepath.Join(config.DataDir, xrayDirName)
}

func xrayBinaryPath() string {
	return filepath.Join(xrayDir(), xrayBinaryName)
}

func xrayConfigPath() string {
	return filepath.Join(xrayDir(), xrayConfigFileName)
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
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(configPath(), append(data, '\n'), 0644)
}

func defaultConfig() *Config {
	return &Config{
		ListenPort: defaultListenPort,
		WSPath:     defaultWSPath,
		Subdomain:  defaultSubdomain,
	}
}
