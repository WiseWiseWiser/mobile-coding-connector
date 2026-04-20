package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// domainConfig is a saved server+token pair.
type domainConfig struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

// agentConfig is the persisted CLI configuration.
type agentConfig struct {
	// Default is the Server field of the entry in Domains to use when
	// no --server flag is supplied. May be empty.
	Default string `json:"default,omitempty"`
	// Domains is the list of configured server endpoints.
	Domains []domainConfig `json:"domains"`

	// Legacy single-domain fields, kept for backward compatibility when
	// reading old config files. They are migrated into Domains on load
	// and are not written out in new files.
	LegacyServer string `json:"server,omitempty"`
	LegacyToken  string `json:"token,omitempty"`
}

// DefaultDomain returns the default-selected domain, or nil if none is set
// (or the default no longer exists in Domains).
func (c *agentConfig) DefaultDomain() *domainConfig {
	if c == nil || c.Default == "" {
		return nil
	}
	for i := range c.Domains {
		if c.Domains[i].Server == c.Default {
			return &c.Domains[i]
		}
	}
	return nil
}

func configFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ai-critic", "remote-agent-config.json"), nil
}

// loadConfig reads the saved agent config, migrating legacy single-domain
// fields into Domains. Missing file returns (nil, nil).
func loadConfig() (*agentConfig, error) {
	path, err := configFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cfg agentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Migrate legacy fields into Domains list.
	if cfg.LegacyServer != "" {
		legacyServer := strings.TrimRight(cfg.LegacyServer, "/")
		if !hasDomain(cfg.Domains, legacyServer) {
			cfg.Domains = append(cfg.Domains, domainConfig{
				Server: legacyServer,
				Token:  cfg.LegacyToken,
			})
		}
		if cfg.Default == "" {
			cfg.Default = legacyServer
		}
		cfg.LegacyServer = ""
		cfg.LegacyToken = ""
	}

	return &cfg, nil
}

func hasDomain(domains []domainConfig, server string) bool {
	for _, d := range domains {
		if d.Server == server {
			return true
		}
	}
	return false
}

func saveConfig(cfg *agentConfig) error {
	path, err := configFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	// Never write legacy fields.
	out := *cfg
	out.LegacyServer = ""
	out.LegacyToken = ""
	if out.Domains == nil {
		out.Domains = []domainConfig{}
	}
	data, err := json.MarshalIndent(&out, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
