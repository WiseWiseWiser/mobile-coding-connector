package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	PortAuthEphemeral = "ephemeral"
	PortAuthCustom    = "custom"
	PortAuthShared    = "shared"
)

// PortMappingAuthFile is the path to persisted port-mapping auth flags.
// The live hop is not restored from this file; it only remembers the last
// --require-auth settings so list/re-add/update can reuse them.
var PortMappingAuthFile = DataDir + "/port-mapping-auth.json"

// PortMappingAuthSecretFile is the cookie-signing secret for mapping hops.
var PortMappingAuthSecretFile = DataDir + "/port-mapping-auth-secret"

// PortAuth is the login hop attached to one mapped port.
type PortAuth struct {
	Required  bool   `json:"required"`
	Ephemeral bool   `json:"ephemeral,omitempty"`
	TokenMode string `json:"tokenMode,omitempty"`
	User      string `json:"user,omitempty"`
	Token     string `json:"token,omitempty"`
}

// Public returns a copy safe for GET /api/ports (no token).
func (a PortAuth) Public() *PortAuth {
	if !a.Required {
		return nil
	}
	out := a
	out.Token = ""
	return &out
}

type portAuthJSON struct {
	Required  bool   `json:"required"`
	Ephemeral bool   `json:"ephemeral,omitempty"`
	TokenMode string `json:"tokenMode,omitempty"`
	User      string `json:"user,omitempty"`
	Token     string `json:"token,omitempty"`
}

// UnmarshalJSON accepts the nested shape and the previous flat requireAuth fields.
func (a *PortAuth) UnmarshalJSON(data []byte) error {
	var w portAuthJSON
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	*a = PortAuth{
		Required:  w.Required,
		Ephemeral: w.Ephemeral,
		TokenMode: w.TokenMode,
		User:      w.User,
		Token:     w.Token,
	}
	if a.Required {
		return nil
	}
	var old struct {
		RequireAuth   bool   `json:"requireAuth"`
		AuthUser      string `json:"authUser"`
		AuthTokenMode string `json:"authTokenMode"`
		AuthToken     string `json:"authToken"`
	}
	if err := json.Unmarshal(data, &old); err != nil {
		return nil
	}
	if !old.RequireAuth {
		return nil
	}
	a.Required = true
	a.User = old.AuthUser
	a.TokenMode = old.AuthTokenMode
	a.Token = old.AuthToken
	return nil
}

// PortMappingAuth maps a port number (as a decimal string) to auth flags.
type PortMappingAuth map[string]PortAuth

func portMappingAuthKey(port int) string {
	return fmt.Sprintf("%d", port)
}

// GenerateEphemeralToken returns 16 random bytes as hex (32 chars).
func GenerateEphemeralToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate mapping token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func normalizePortAuth(entry *PortAuth) {
	if entry == nil {
		return
	}
	entry.User = strings.TrimSpace(entry.User)
	entry.Token = strings.TrimSpace(entry.Token)
	entry.TokenMode = strings.ToLower(strings.TrimSpace(entry.TokenMode))
	switch entry.TokenMode {
	case PortAuthEphemeral:
		entry.Ephemeral = true
	case PortAuthCustom:
		entry.Ephemeral = false
	case PortAuthShared:
		entry.Ephemeral = false
		entry.Token = ""
	default:
		if entry.Ephemeral || entry.Token != "" {
			if entry.Ephemeral {
				entry.TokenMode = PortAuthEphemeral
			} else {
				entry.TokenMode = PortAuthCustom
			}
		}
	}
}

// LoadPortMappingAuth loads persisted mapping auth flags.
func LoadPortMappingAuth() (PortMappingAuth, error) {
	data, err := os.ReadFile(PortMappingAuthFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(PortMappingAuth), nil
		}
		return nil, fmt.Errorf("failed to read port mapping auth file: %w", err)
	}
	var mappings PortMappingAuth
	if err := json.Unmarshal(data, &mappings); err != nil {
		return nil, fmt.Errorf("failed to parse port mapping auth file: %w", err)
	}
	if mappings == nil {
		mappings = make(PortMappingAuth)
	}
	return mappings, nil
}

func savePortMappingAuth(mappings PortMappingAuth) error {
	if err := os.MkdirAll(DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	data, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal port mapping auth file: %w", err)
	}
	if err := os.WriteFile(PortMappingAuthFile, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("failed to write port mapping auth file: %w", err)
	}
	return nil
}

// GetPortMappingAuth returns the saved auth flags for a port, or nil when none.
func GetPortMappingAuth(port int) (*PortAuth, error) {
	mappings, err := LoadPortMappingAuth()
	if err != nil {
		return nil, err
	}
	entry, ok := mappings[portMappingAuthKey(port)]
	if !ok || !entry.Required {
		return nil, nil
	}
	normalizePortAuth(&entry)
	return &entry, nil
}

// SetPortMappingAuth saves auth flags for a port. A non-required entry
// deletes any saved flags instead of storing a disabled record.
func SetPortMappingAuth(port int, entry PortAuth) error {
	if !entry.Required {
		return DeletePortMappingAuth(port)
	}
	normalizePortAuth(&entry)
	mappings, err := LoadPortMappingAuth()
	if err != nil {
		return err
	}
	mappings[portMappingAuthKey(port)] = entry
	return savePortMappingAuth(mappings)
}

// DeletePortMappingAuth forgets saved auth flags for a port.
func DeletePortMappingAuth(port int) error {
	mappings, err := LoadPortMappingAuth()
	if err != nil {
		return err
	}
	key := portMappingAuthKey(port)
	if _, ok := mappings[key]; !ok {
		return nil
	}
	delete(mappings, key)
	return savePortMappingAuth(mappings)
}
