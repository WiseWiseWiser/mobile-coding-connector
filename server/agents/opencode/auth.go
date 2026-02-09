// Package opencode provides authentication status checking for OpenCode.
package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AuthProvider represents an authenticated provider in OpenCode.
type AuthProvider struct {
	Name         string `json:"name"`
	HasAPIKey    bool   `json:"has_api_key"` // Indicates whether an API key is configured
}

// AuthStatus represents the authentication status for OpenCode.
type AuthStatus struct {
	Authenticated bool           `json:"authenticated"`
	Providers     []AuthProvider `json:"providers"`
	ConfigPath    string         `json:"config_path"`
}

// GetAuthStatus reads the OpenCode auth.json file and returns the authentication status.
func GetAuthStatus() (*AuthStatus, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return &AuthStatus{Authenticated: false}, nil
	}

	authPath := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	status := &AuthStatus{
		ConfigPath: authPath,
	}

	data, err := os.ReadFile(authPath)
	if err != nil {
		// File doesn't exist or can't be read - not authenticated
		return status, nil
	}

	// Parse the auth.json file
	// The format is typically: {"providers": {"provider_name": {"api_key": "..."}}}
	var authData map[string]interface{}
	if err := json.Unmarshal(data, &authData); err != nil {
		return status, nil
	}

	// Check for providers
	if providers, ok := authData["providers"].(map[string]interface{}); ok {
		for name, providerData := range providers {
			provider := AuthProvider{Name: name}
			// Check if there's an API key configured
			if pd, ok := providerData.(map[string]interface{}); ok {
				if _, hasKey := pd["api_key"]; hasKey {
					provider.HasAPIKey = true
				}
			}
			status.Providers = append(status.Providers, provider)
		}
	}

	status.Authenticated = len(status.Providers) > 0
	return status, nil
}
