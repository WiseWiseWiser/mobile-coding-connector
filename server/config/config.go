package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the application configuration
type Config struct {
	// AI configuration
	AI AIConfig `json:"ai"`

	// PortForwarding configuration
	PortForwarding PortForwardingConfig `json:"port_forwarding,omitempty"`

	// Server configuration
	Server ServerConfig `json:"server,omitempty"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	// ProjectDir is the explicitly configured project directory.
	// When set, this overrides the auto-detected project directory.
	ProjectDir string `json:"project_dir,omitempty"`
}

// PortForwardingConfig represents the port forwarding configuration
type PortForwardingConfig struct {
	// Providers is a list of tunnel provider configurations
	Providers []PortForwardProviderConfig `json:"providers,omitempty"`
}

// PortForwardProviderConfig represents a single tunnel provider configuration
type PortForwardProviderConfig struct {
	// Type is the provider type: "localtunnel", "cloudflare_quick", or "cloudflare_tunnel"
	Type string `json:"type"`

	// Enabled controls whether this provider is available (default: true)
	Enabled *bool `json:"enabled,omitempty"`

	// Cloudflare tunnel specific settings (only for type "cloudflare_tunnel")
	Cloudflare *CloudflareTunnelConfig `json:"cloudflare,omitempty"`
}

// CloudflareTunnelConfig holds config for a named Cloudflare tunnel
type CloudflareTunnelConfig struct {
	// TunnelName is the named tunnel identifier (e.g. "port-forward-tunnel").
	// Required. If the tunnel doesn't exist, it will be created automatically.
	TunnelName string `json:"tunnel_name,omitempty"`

	// TunnelID is the UUID of the tunnel. Optional - resolved automatically
	// from TunnelName if not specified.
	TunnelID string `json:"tunnel_id,omitempty"`

	// BaseDomain is the base domain under which random subdomains are created
	// (e.g. "xhd2015.xyz" -> generates "brave-lake-fern.xhd2015.xyz").
	// Required.
	BaseDomain string `json:"base_domain"`

	// ConfigPath is the directory for the port-forward config file.
	// Default: ~/.cloudflared
	ConfigPath string `json:"config_path,omitempty"`

	// CredentialsFile is the path to the tunnel credentials JSON file.
	// Optional - resolved automatically from TunnelID if not specified.
	CredentialsFile string `json:"credentials_file,omitempty"`
}

// IsEnabled returns whether a provider config is enabled (default true)
func (p *PortForwardProviderConfig) IsEnabled() bool {
	if p.Enabled == nil {
		return true
	}
	return *p.Enabled
}

// AIConfig represents the AI configuration
type AIConfig struct {
	// Providers is a list of AI provider configurations
	Providers []ProviderConfig `json:"providers"`

	// Models is a list of available models with their provider mapping
	Models []ModelConfig `json:"models"`

	// DefaultProvider is the default provider to use
	DefaultProvider string `json:"default_provider,omitempty"`

	// DefaultModel is the default model to use
	DefaultModel string `json:"default_model,omitempty"`
}

// ProviderConfig represents an AI provider configuration
type ProviderConfig struct {
	// Name is the unique identifier for this provider (e.g., "deepseek", "moonshot-cn", "openai")
	Name string `json:"name"`

	// BaseURL is the API endpoint for this provider
	BaseURL string `json:"base_url"`

	// APIKey is the API key for this provider
	APIKey string `json:"api_key,omitempty"`
}

// ModelConfig represents an AI model configuration
type ModelConfig struct {
	// Provider is the name of the provider this model belongs to
	Provider string `json:"provider"`

	// Model is the model identifier (e.g., "deepseek-reasoner", "kimi-k2")
	Model string `json:"model"`

	// DisplayName is a human-readable name for the model (optional)
	DisplayName string `json:"display_name,omitempty"`

	// MaxTokens is the max tokens for this model (optional)
	MaxTokens int `json:"max_tokens,omitempty"`
}

// global config instance
var globalConfig *Config

// Load loads configuration from a JSON file
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	globalConfig = &cfg
	return &cfg, nil
}

// Get returns the global config instance
// Returns nil if config is not loaded
func Get() *Config {
	return globalConfig
}

// Set sets the global config instance
func Set(cfg *Config) {
	globalConfig = cfg
}

// GetAI returns the AI configuration
func (c *Config) GetAI() AIConfig {
	return c.AI
}

// GetProvider returns a provider config by name
func (c *Config) GetProvider(name string) *ProviderConfig {
	for i := range c.AI.Providers {
		if c.AI.Providers[i].Name == name {
			return &c.AI.Providers[i]
		}
	}
	return nil
}

// GetModel returns a model config by provider and model name
func (c *Config) GetModel(provider, model string) *ModelConfig {
	for i := range c.AI.Models {
		if c.AI.Models[i].Provider == provider && c.AI.Models[i].Model == model {
			return &c.AI.Models[i]
		}
	}
	return nil
}

// GetDefaultAIConfig returns the default AI configuration for making API calls
func (c *Config) GetDefaultAIConfig() (baseURL, apiKey, model string) {
	// Use default provider/model if specified
	providerName := c.AI.DefaultProvider
	modelName := c.AI.DefaultModel

	// Fall back to first provider/model if not specified
	if providerName == "" && len(c.AI.Providers) > 0 {
		providerName = c.AI.Providers[0].Name
	}
	if modelName == "" && len(c.AI.Models) > 0 {
		modelName = c.AI.Models[0].Model
	}

	provider := c.GetProvider(providerName)
	if provider != nil {
		baseURL = provider.BaseURL
		apiKey = provider.APIKey
	}

	model = modelName
	return
}

// GetModelsForProvider returns all models for a given provider
func (c *Config) GetModelsForProvider(provider string) []ModelConfig {
	var models []ModelConfig
	for _, m := range c.AI.Models {
		if m.Provider == provider {
			models = append(models, m)
		}
	}
	return models
}

// GetAvailableProviders returns a list of configured providers
func (c *Config) GetAvailableProviders() []ProviderConfig {
	return c.AI.Providers
}

// GetAvailableModels returns a list of all configured models
func (c *Config) GetAvailableModels() []ModelConfig {
	return c.AI.Models
}

// ServerProjectConfig represents the server project configuration stored in .ai-critic/server-project.json
type ServerProjectConfig struct {
	ProjectDir string `json:"project_dir,omitempty"`
}

// LoadServerProjectConfig loads the server project configuration from .ai-critic/server-project.json
func LoadServerProjectConfig() (*ServerProjectConfig, error) {
	data, err := os.ReadFile(ServerProjectFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &ServerProjectConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read server project config: %w", err)
	}

	var cfg ServerProjectConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse server project config: %w", err)
	}

	return &cfg, nil
}

// SaveServerProjectConfig saves the server project configuration to .ai-critic/server-project.json
func SaveServerProjectConfig(cfg *ServerProjectConfig) error {
	// Ensure directory exists
	if err := os.MkdirAll(DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal server project config: %w", err)
	}

	if err := os.WriteFile(ServerProjectFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write server project config: %w", err)
	}

	return nil
}

// GetServerProjectDir returns the configured server project directory
// Returns empty string if not configured
func GetServerProjectDir() string {
	cfg, err := LoadServerProjectConfig()
	if err != nil {
		return ""
	}
	return cfg.ProjectDir
}

// SetServerProjectDir sets and saves the server project directory
func SetServerProjectDir(projectDir string) error {
	cfg, err := LoadServerProjectConfig()
	if err != nil {
		cfg = &ServerProjectConfig{}
	}
	cfg.ProjectDir = projectDir
	return SaveServerProjectConfig(cfg)
}
