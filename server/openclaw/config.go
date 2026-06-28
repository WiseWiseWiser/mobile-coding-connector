package openclaw

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/ai-critic/server/config"
)

const (
	defaultGatewayPort = 18789
	configFileName     = "openclaw.json"
	stateFileName      = "state.json"
	generatedConfig    = "openclaw.json"
	slackModeSocket    = "socket"
)

var _testDataDir string

type SlackConfig struct {
	Enabled        bool     `json:"enabled"`
	Mode           string   `json:"mode"`
	BotToken       string   `json:"bot_token,omitempty"`
	AppToken       string   `json:"app_token,omitempty"`
	DMPolicy       string   `json:"dm_policy,omitempty"`
	AllowFrom      []string `json:"allow_from,omitempty"`
	RequireMention *bool    `json:"require_mention,omitempty"`
}

type Config struct {
	Enabled     bool         `json:"enabled"`
	GatewayPort int          `json:"gateway_port"`
	Workspace   string       `json:"workspace,omitempty"`
	AutoStart   bool         `json:"auto_start"`
	Model       string       `json:"model,omitempty"`
	Slack       *SlackConfig `json:"slack,omitempty"`
}

type RuntimeState struct {
	Running   bool   `json:"running"`
	StartedAt string `json:"started_at,omitempty"`
	MockPID   int    `json:"mock_pid,omitempty"`
	LastError string `json:"last_error,omitempty"`
	Mocked    bool   `json:"mocked"`
}

func dataDir() string {
	if _testDataDir != "" {
		return _testDataDir
	}
	return config.DataDir
}

func configPath() string {
	return filepath.Join(dataDir(), configFileName)
}

func openclawDir() string {
	return filepath.Join(dataDir(), "openclaw")
}

func generatedConfigPath() string {
	return filepath.Join(openclawDir(), generatedConfig)
}

func statePath() string {
	return filepath.Join(openclawDir(), stateFileName)
}

func defaultConfig() *Config {
	return &Config{
		GatewayPort: defaultGatewayPort,
	}
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
	normalizeConfig(cfg)
	return cfg, nil
}

func SaveConfig(cfg *Config) error {
	normalizeConfig(cfg)
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func LoadState() (*RuntimeState, error) {
	data, err := os.ReadFile(statePath())
	if os.IsNotExist(err) {
		return &RuntimeState{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read state: %w", err)
	}
	state := &RuntimeState{}
	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("failed to parse state: %w", err)
	}
	return state, nil
}

func SaveState(state *RuntimeState) error {
	if state == nil {
		state = &RuntimeState{}
	}
	path := statePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create state dir: %w", err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func normalizeConfig(cfg *Config) {
	if cfg == nil {
		return
	}
	if cfg.GatewayPort == 0 {
		cfg.GatewayPort = defaultGatewayPort
	}
	if cfg.Slack != nil {
		if cfg.Slack.Mode == "" {
			cfg.Slack.Mode = slackModeSocket
		}
		if cfg.Slack.DMPolicy == "" {
			cfg.Slack.DMPolicy = "pairing"
		}
		if cfg.Slack.RequireMention == nil {
			v := true
			cfg.Slack.RequireMention = &v
		}
	}
}

func MaskConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}
	masked := *cfg
	if masked.Slack != nil {
		slack := *masked.Slack
		slack.BotToken = maskSecret(slack.BotToken)
		slack.AppToken = maskSecret(slack.AppToken)
		masked.Slack = &slack
	}
	return &masked
}

func maskSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "***"
}

func MergeConfig(base *Config, update ConfigUpdate) *Config {
	if base == nil {
		base = defaultConfig()
	}
	merged := *base

	if update.Enabled != nil {
		merged.Enabled = *update.Enabled
	}
	if update.GatewayPort != nil {
		merged.GatewayPort = *update.GatewayPort
	}
	if update.Workspace != "" {
		merged.Workspace = update.Workspace
	}
	if update.AutoStart != nil {
		merged.AutoStart = *update.AutoStart
	}
	if update.Model != "" {
		merged.Model = update.Model
	}
	if update.Slack != nil {
		merged.Slack = mergeSlackConfig(merged.Slack, *update.Slack)
	}

	normalizeConfig(&merged)
	return &merged
}

type ConfigUpdate struct {
	Enabled     *bool         `json:"enabled,omitempty"`
	GatewayPort *int          `json:"gateway_port,omitempty"`
	Workspace   string        `json:"workspace,omitempty"`
	AutoStart   *bool         `json:"auto_start,omitempty"`
	Model       string        `json:"model,omitempty"`
	Slack       *SlackUpdate  `json:"slack,omitempty"`
}

type SlackUpdate struct {
	Enabled        *bool    `json:"enabled,omitempty"`
	Mode           string   `json:"mode,omitempty"`
	BotToken       string   `json:"bot_token,omitempty"`
	AppToken       string   `json:"app_token,omitempty"`
	DMPolicy       string   `json:"dm_policy,omitempty"`
	AllowFrom      []string `json:"allow_from,omitempty"`
	RequireMention *bool    `json:"require_mention,omitempty"`
}

func mergeSlackConfig(base *SlackConfig, update SlackUpdate) *SlackConfig {
	var merged SlackConfig
	if base != nil {
		merged = *base
	}
	if update.Enabled != nil {
		merged.Enabled = *update.Enabled
	}
	if update.Mode != "" {
		merged.Mode = update.Mode
	}
	if update.BotToken != "" {
		merged.BotToken = update.BotToken
	}
	if update.AppToken != "" {
		merged.AppToken = update.AppToken
	}
	if update.DMPolicy != "" {
		merged.DMPolicy = update.DMPolicy
	}
	if update.AllowFrom != nil {
		merged.AllowFrom = update.AllowFrom
	}
	if update.RequireMention != nil {
		merged.RequireMention = update.RequireMention
	}
	normalizeConfig(&Config{Slack: &merged})
	return &merged
}

func ValidateStartConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}
	if cfg.Slack != nil && cfg.Slack.Enabled {
		if strings.TrimSpace(cfg.Slack.BotToken) == "" {
			return fmt.Errorf("slack bot token required when slack is enabled")
		}
		mode := cfg.Slack.Mode
		if mode == "" {
			mode = slackModeSocket
		}
		if mode != slackModeSocket {
			return fmt.Errorf("only socket mode is supported (got %q)", mode)
		}
		if strings.TrimSpace(cfg.Slack.AppToken) == "" {
			return fmt.Errorf("slack app token required for socket mode")
		}
	}
	return nil
}