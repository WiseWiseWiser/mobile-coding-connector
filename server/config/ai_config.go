package config

import (
	"fmt"
)

// Global AI models config instance
var globalAIModelsConfig *AIModelsConfig

// LoadAIConfig loads AI configuration from the new file if it exists,
// otherwise from the legacy config. This should be called after Load().
func LoadAIConfig(legacyCfg *Config) (*AIModelsConfig, error) {
	// Check if new AI models file exists
	if AIModelsFileExists() {
		cfg, err := LoadAIModelsConfig()
		if err != nil {
			return nil, err
		}
		globalAIModelsConfig = cfg
		return cfg, nil
	}

	// Fall back to legacy config
	if legacyCfg != nil {
		cfg := &AIModelsConfig{
			Providers:       legacyCfg.AI.Providers,
			Models:          legacyCfg.AI.Models,
			DefaultProvider: legacyCfg.AI.DefaultProvider,
			DefaultModel:    legacyCfg.AI.DefaultModel,
		}
		globalAIModelsConfig = cfg
		return cfg, nil
	}

	// Return empty config
	cfg := &AIModelsConfig{
		Providers: []ProviderConfig{},
		Models:    []ModelConfig{},
	}
	globalAIModelsConfig = cfg
	return cfg, nil
}

// GetAIConfig returns the global AI models config instance.
// Returns nil if not loaded.
func GetAIConfig() *AIModelsConfig {
	return globalAIModelsConfig
}

// SetAIConfig sets the global AI models config instance.
func SetAIConfig(cfg *AIModelsConfig) {
	globalAIModelsConfig = cfg
}

// ConfigAdapter wraps AIModelsConfig to provide the same interface as Config
type ConfigAdapter struct {
	ai AIMConfig
}

// AIMConfig wraps the AI configuration for adapter
type AIMConfig struct {
	Providers       []ProviderConfig
	Models          []ModelConfig
	DefaultProvider string
	DefaultModel    string
}

// NewConfigAdapter creates a new ConfigAdapter from AIModelsConfig
func NewConfigAdapter(cfg *AIModelsConfig) *ConfigAdapter {
	if cfg == nil {
		return &ConfigAdapter{ai: AIMConfig{}}
	}
	return &ConfigAdapter{
		ai: AIMConfig{
			Providers:       cfg.Providers,
			Models:          cfg.Models,
			DefaultProvider: cfg.DefaultProvider,
			DefaultModel:    cfg.DefaultModel,
		},
	}
}

// GetProvider returns a provider config by name
func (c *ConfigAdapter) GetProvider(name string) *ProviderConfig {
	for i := range c.ai.Providers {
		if c.ai.Providers[i].Name == name {
			return &c.ai.Providers[i]
		}
	}
	return nil
}

// GetModel returns a model config by provider and model name
func (c *ConfigAdapter) GetModel(provider, model string) *ModelConfig {
	for i := range c.ai.Models {
		if c.ai.Models[i].Provider == provider && c.ai.Models[i].Model == model {
			return &c.ai.Models[i]
		}
	}
	return nil
}

// GetDefaultAIConfig returns the default AI configuration for making API calls
func (c *ConfigAdapter) GetDefaultAIConfig() (baseURL, apiKey, model string) {
	providerName := c.ai.DefaultProvider
	modelName := c.ai.DefaultModel

	// Fall back to first provider/model if not specified
	if providerName == "" && len(c.ai.Providers) > 0 {
		providerName = c.ai.Providers[0].Name
	}
	if modelName == "" && len(c.ai.Models) > 0 {
		modelName = c.ai.Models[0].Model
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
func (c *ConfigAdapter) GetModelsForProvider(provider string) []ModelConfig {
	var models []ModelConfig
	for _, m := range c.ai.Models {
		if m.Provider == provider {
			models = append(models, m)
		}
	}
	return models
}

// GetAvailableProviders returns a list of configured providers
func (c *ConfigAdapter) GetAvailableProviders() []ProviderConfig {
	return c.ai.Providers
}

// GetAvailableModels returns a list of all configured models
func (c *ConfigAdapter) GetAvailableModels() []ModelConfig {
	return c.ai.Models
}

// GetDefaultProvider returns the default provider name
func (c *ConfigAdapter) GetDefaultProvider() string {
	return c.ai.DefaultProvider
}

// GetDefaultModel returns the default model name
func (c *ConfigAdapter) GetDefaultModel() string {
	return c.ai.DefaultModel
}

// ToConfigResponse converts the adapter to a response format for the frontend
func (c *ConfigAdapter) ToConfigResponse() map[string]interface{} {
	return map[string]interface{}{
		"providers":        c.ai.Providers,
		"models":           c.ai.Models,
		"default_provider": c.ai.DefaultProvider,
		"default_model":    c.ai.DefaultModel,
	}
}

// UpdateFromRequest updates the config from a frontend request
func (c *ConfigAdapter) UpdateFromRequest(req map[string]interface{}) error {
	if providers, ok := req["providers"].([]interface{}); ok {
		c.ai.Providers = make([]ProviderConfig, 0, len(providers))
		for _, p := range providers {
			if provMap, ok := p.(map[string]interface{}); ok {
				prov := ProviderConfig{
					Name:    getString(provMap, "name"),
					BaseURL: getString(provMap, "base_url"),
					APIKey:  getString(provMap, "api_key"),
				}
				c.ai.Providers = append(c.ai.Providers, prov)
			}
		}
	}

	if models, ok := req["models"].([]interface{}); ok {
		c.ai.Models = make([]ModelConfig, 0, len(models))
		for _, m := range models {
			if modelMap, ok := m.(map[string]interface{}); ok {
				model := ModelConfig{
					Provider:    getString(modelMap, "provider"),
					Model:       getString(modelMap, "model"),
					DisplayName: getString(modelMap, "display_name"),
				}
				if maxTokens, ok := modelMap["max_tokens"].(float64); ok {
					model.MaxTokens = int(maxTokens)
				}
				c.ai.Models = append(c.ai.Models, model)
			}
		}
	}

	c.ai.DefaultProvider = getString(req, "default_provider")
	c.ai.DefaultModel = getString(req, "default_model")

	return nil
}

// ToAIModelsConfig converts the adapter back to AIModelsConfig
func (c *ConfigAdapter) ToAIModelsConfig() *AIModelsConfig {
	return &AIModelsConfig{
		Providers:       c.ai.Providers,
		Models:          c.ai.Models,
		DefaultProvider: c.ai.DefaultProvider,
		DefaultModel:    c.ai.DefaultModel,
	}
}

// getString safely gets a string value from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// GetEffectiveAIConfig returns the effective AI configuration.
// It checks the new AI models file first, then falls back to legacy config.
func GetEffectiveAIConfig(legacyCfg *Config) (*ConfigAdapter, error) {
	// Check if new AI models file exists
	if AIModelsFileExists() {
		cfg, err := LoadAIModelsConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load AI models config: %w", err)
		}
		return NewConfigAdapter(cfg), nil
	}

	// Fall back to legacy config
	if legacyCfg != nil {
		return NewConfigAdapter(&AIModelsConfig{
			Providers:       legacyCfg.AI.Providers,
			Models:          legacyCfg.AI.Models,
			DefaultProvider: legacyCfg.AI.DefaultProvider,
			DefaultModel:    legacyCfg.AI.DefaultModel,
		}), nil
	}

	// Return empty adapter
	return NewConfigAdapter(&AIModelsConfig{
		Providers: []ProviderConfig{},
		Models:    []ModelConfig{},
	}), nil
}
