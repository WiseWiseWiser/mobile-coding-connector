package exposed_opencode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func authJSONPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "opencode", "auth.json")
}

// AuthProvider represents an authenticated provider in OpenCode.
type AuthProvider struct {
	Name      string `json:"name"`
	HasAPIKey bool   `json:"has_api_key"`
}

// AuthStatus represents the authentication status for OpenCode.
type AuthStatus struct {
	Authenticated bool           `json:"authenticated"`
	Providers     []AuthProvider `json:"providers"`
	ConfigPath    string         `json:"config_path"`
}

// authEntry is the on-disk format for a provider entry in auth.json.
type authEntry struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

func readAuthFile() (map[string]authEntry, error) {
	data, err := os.ReadFile(authJSONPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries map[string]authEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func writeAuthFile(entries map[string]authEntry) error {
	path := authJSONPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// GetAuthStatus reads the OpenCode auth.json file and returns the authentication status.
func GetAuthStatus() (*AuthStatus, error) {
	status := &AuthStatus{
		ConfigPath: authJSONPath(),
	}
	entries, err := readAuthFile()
	if err != nil {
		return status, nil
	}
	for name, entry := range entries {
		status.Providers = append(status.Providers, AuthProvider{
			Name:      name,
			HasAPIKey: entry.Key != "",
		})
	}
	sort.Slice(status.Providers, func(i, j int) bool {
		return status.Providers[i].Name < status.Providers[j].Name
	})
	status.Authenticated = len(status.Providers) > 0
	return status, nil
}

// AuthKeyEntry is returned to clients with the key masked for security.
type AuthKeyEntry struct {
	Provider  string `json:"provider"`
	Type      string `json:"type"`
	MaskedKey string `json:"masked_key"`
}

// GetAuthKeys returns all provider keys with values masked.
func GetAuthKeys() ([]AuthKeyEntry, error) {
	entries, err := readAuthFile()
	if err != nil {
		return nil, err
	}
	var result []AuthKeyEntry
	for name, entry := range entries {
		result = append(result, AuthKeyEntry{
			Provider:  name,
			Type:      entry.Type,
			MaskedKey: maskKey(entry.Key),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Provider < result[j].Provider
	})
	return result, nil
}

// SetAuthKey sets or updates a provider API key.
func SetAuthKey(provider, key string) error {
	if provider == "" {
		return fmt.Errorf("provider is required")
	}
	entries, err := readAuthFile()
	if err != nil {
		return err
	}
	if entries == nil {
		entries = make(map[string]authEntry)
	}
	entries[provider] = authEntry{Type: "api", Key: key}
	return writeAuthFile(entries)
}

// DeleteAuthKey removes a provider from auth.json.
func DeleteAuthKey(provider string) error {
	entries, err := readAuthFile()
	if err != nil {
		return err
	}
	if entries == nil {
		return nil
	}
	delete(entries, provider)
	return writeAuthFile(entries)
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
