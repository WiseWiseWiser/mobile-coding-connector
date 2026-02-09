package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

// AgentConfig holds the configuration for a single agent
type AgentConfig struct {
	// BinaryPath is the custom path to the agent binary (optional)
	BinaryPath string `json:"binary_path,omitempty"`
}

// AgentsConfig holds all agent configurations
type AgentsConfig struct {
	// Agents maps agent ID to its config
	Agents map[string]AgentConfig `json:"agents"`
}

var (
	configMu    sync.RWMutex
	agentConfig *AgentsConfig
)

// configPath returns the path to the agents.json config file
func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, config.AgentsFile)
}

// LoadConfig loads the agents configuration from disk
func LoadConfig() (*AgentsConfig, error) {
	configMu.RLock()
	if agentConfig != nil {
		cfg := *agentConfig
		configMu.RUnlock()
		return &cfg, nil
	}
	configMu.RUnlock()

	configMu.Lock()
	defer configMu.Unlock()

	// Double-check after acquiring write lock
	if agentConfig != nil {
		cfg := *agentConfig
		return &cfg, nil
	}

	path := configPath()
	if path == "" {
		agentConfig = &AgentsConfig{Agents: make(map[string]AgentConfig)}
		return agentConfig, nil
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		agentConfig = &AgentsConfig{Agents: make(map[string]AgentConfig)}
		return agentConfig, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg AgentsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Agents == nil {
		cfg.Agents = make(map[string]AgentConfig)
	}

	agentConfig = &cfg
	return &cfg, nil
}

// SaveConfig saves the agents configuration to disk
func SaveConfig(cfg *AgentsConfig) error {
	configMu.Lock()
	defer configMu.Unlock()

	path := configPath()
	if path == "" {
		return nil
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}

	agentConfig = cfg
	return nil
}

// GetAgentBinaryPath returns the custom binary path for an agent, or empty string if not configured
func GetAgentBinaryPath(agentID string) string {
	cfg, err := LoadConfig()
	if err != nil {
		return ""
	}
	if ac, ok := cfg.Agents[agentID]; ok {
		return ac.BinaryPath
	}
	return ""
}

// SetAgentBinaryPath sets the custom binary path for an agent
func SetAgentBinaryPath(agentID, binaryPath string) error {
	cfg, err := LoadConfig()
	if err != nil {
		cfg = &AgentsConfig{Agents: make(map[string]AgentConfig)}
	}

	if binaryPath == "" {
		delete(cfg.Agents, agentID)
	} else {
		cfg.Agents[agentID] = AgentConfig{BinaryPath: binaryPath}
	}

	return SaveConfig(cfg)
}

// InvalidateConfigCache clears the cached config so it will be reloaded on next access
func InvalidateConfigCache() {
	configMu.Lock()
	defer configMu.Unlock()
	agentConfig = nil
}
