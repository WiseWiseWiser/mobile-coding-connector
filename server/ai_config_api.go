package server

import (
	"encoding/json"
	"net/http"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

// registerAIConfigAPI registers the AI config management API endpoints
func registerAIConfigAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/ai-config", handleAIConfig)
}

// handleAIConfig handles GET and POST requests for AI configuration
func handleAIConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetAIConfig(w, r)
	case http.MethodPost:
		handleSaveAIConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// AIProviderResponse represents a provider in the API response
type AIProviderResponse struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key,omitempty"`
}

// AIModelResponse represents a model in the API response
type AIModelResponse struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	DisplayName string `json:"display_name,omitempty"`
	MaxTokens   int    `json:"max_tokens,omitempty"`
}

// AIConfigResponse represents the full AI config response
type AIConfigResponse struct {
	Providers       []AIProviderResponse `json:"providers"`
	Models          []AIModelResponse    `json:"models"`
	DefaultProvider string               `json:"default_provider,omitempty"`
	DefaultModel    string               `json:"default_model,omitempty"`
	UsingNewFile    bool                 `json:"using_new_file"`
}

// handleGetAIConfig returns the current AI configuration
func handleGetAIConfig(w http.ResponseWriter, r *http.Request) {
	// Load from the new file if it exists, otherwise from adapter
	var cfg *config.AIModelsConfig
	var err error

	if config.AIModelsFileExists() {
		cfg, err = config.LoadAIModelsConfig()
	} else {
		// Fall back to adapter/legacy
		adapter := getEffectiveAIConfig()
		if adapter != nil {
			cfg = adapter.ToAIModelsConfig()
		} else {
			cfg = &config.AIModelsConfig{
				Providers: []config.ProviderConfig{},
				Models:    []config.ModelConfig{},
			}
		}
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Convert to response format
	resp := AIConfigResponse{
		Providers:       make([]AIProviderResponse, 0, len(cfg.Providers)),
		Models:          make([]AIModelResponse, 0, len(cfg.Models)),
		DefaultProvider: cfg.DefaultProvider,
		DefaultModel:    cfg.DefaultModel,
		UsingNewFile:    config.AIModelsFileExists(),
	}

	for _, p := range cfg.Providers {
		resp.Providers = append(resp.Providers, AIProviderResponse{
			Name:    p.Name,
			BaseURL: p.BaseURL,
			APIKey:  p.APIKey,
		})
	}

	for _, m := range cfg.Models {
		resp.Models = append(resp.Models, AIModelResponse{
			Provider:    m.Provider,
			Model:       m.Model,
			DisplayName: m.DisplayName,
			MaxTokens:   m.MaxTokens,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// SaveAIConfigRequest represents the request body for saving AI config
type SaveAIConfigRequest struct {
	Providers       []AIProviderRequest `json:"providers"`
	Models          []AIModelRequest    `json:"models"`
	DefaultProvider string              `json:"default_provider,omitempty"`
	DefaultModel    string              `json:"default_model,omitempty"`
}

// AIProviderRequest represents a provider in the save request
type AIProviderRequest struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

// AIModelRequest represents a model in the save request
type AIModelRequest struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	DisplayName string `json:"display_name,omitempty"`
	MaxTokens   int    `json:"max_tokens,omitempty"`
}

// handleSaveAIConfig saves the AI configuration to the new file
func handleSaveAIConfig(w http.ResponseWriter, r *http.Request) {
	var req SaveAIConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Convert request to config
	cfg := &config.AIModelsConfig{
		Providers:       make([]config.ProviderConfig, 0, len(req.Providers)),
		Models:          make([]config.ModelConfig, 0, len(req.Models)),
		DefaultProvider: req.DefaultProvider,
		DefaultModel:    req.DefaultModel,
	}

	for _, p := range req.Providers {
		cfg.Providers = append(cfg.Providers, config.ProviderConfig{
			Name:    p.Name,
			BaseURL: p.BaseURL,
			APIKey:  p.APIKey,
		})
	}

	for _, m := range req.Models {
		cfg.Models = append(cfg.Models, config.ModelConfig{
			Provider:    m.Provider,
			Model:       m.Model,
			DisplayName: m.DisplayName,
			MaxTokens:   m.MaxTokens,
		})
	}

	// Save to new file
	if err := config.SaveAIModelsConfig(cfg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Update in-memory config
	newAdapter := config.NewConfigAdapter(cfg)
	SetAIConfigAdapter(newAdapter)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
