// Package localauth resolves Bearer tokens for the local macOS menu-bar app.
// Resolution order: local-agent-config.json → server-credentials → none.
// Fall-through only on missing/empty/invalid sources (not HTTP 401).
package localauth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// TokenSource identifies where ResolveLocalServerToken obtained the token.
type TokenSource string

const (
	SourceConfig      TokenSource = "config"
	SourceCredentials TokenSource = "credentials"
	SourceNone        TokenSource = "none"
)

// Options controls where resolve looks for files.
// DataDir empty defaults to ~/.ai-critic.
// ConfigPath / CredentialsPath override the default file paths under DataDir when set.
type Options struct {
	DataDir         string
	ConfigPath      string
	CredentialsPath string
}

// configFile is the on-disk local-agent CLI schema.
type configFile struct {
	Default string         `json:"default"`
	Domains []configDomain `json:"domains"`
}

type configDomain struct {
	Server string `json:"server"`
	Token  string `json:"token"`
}

// Local loopback servers the menu-bar app targets (after normalize).
var localLoopbackServers = []string{
	"http://localhost:23712",
	"http://127.0.0.1:23712",
}

// ResolveLocalServerToken returns a bearer token and its source for the local
// loopback server. Missing files, invalid JSON, and empty tokens fall through;
// resolve never returns an error.
func ResolveLocalServerToken(opts Options) (token string, source TokenSource) {
	dataDir := opts.DataDir
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			// Cannot resolve home; still try credentials path if absolute overrides set.
			dataDir = ""
		} else {
			dataDir = filepath.Join(home, ".ai-critic")
		}
	}

	configPath := opts.ConfigPath
	if configPath == "" {
		if dataDir != "" {
			configPath = filepath.Join(dataDir, "local-agent-config.json")
		}
	}
	credsPath := opts.CredentialsPath
	if credsPath == "" {
		if dataDir != "" {
			credsPath = filepath.Join(dataDir, "server-credentials")
		}
	}

	if configPath != "" {
		if t, ok := tokenFromConfig(configPath); ok {
			return t, SourceConfig
		}
	}
	if credsPath != "" {
		if t, ok := tokenFromCredentials(credsPath); ok {
			return t, SourceCredentials
		}
	}
	return "", SourceNone
}

// AuthorizationHeader formats a Bearer token header value.
// Empty token returns "" (caller should omit the header).
func AuthorizationHeader(token string) string {
	if token == "" {
		return ""
	}
	return "Bearer " + token
}

// NormalizeServer trims space and trailing slashes (same contract as remoteconfig).
func NormalizeServer(server string) string {
	return strings.TrimRight(strings.TrimSpace(server), "/")
}

func tokenFromConfig(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var cfg configFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", false
	}

	// 1) Prefer a local loopback domain with a non-empty trimmed token.
	for _, target := range localLoopbackServers {
		if t, ok := domainTokenMatching(cfg.Domains, target); ok {
			return t, true
		}
	}

	// 2) Else use the domain matching default (after normalize).
	def := NormalizeServer(cfg.Default)
	if def != "" {
		if t, ok := domainTokenMatching(cfg.Domains, def); ok {
			return t, true
		}
	}
	return "", false
}

func domainTokenMatching(domains []configDomain, wantNormalized string) (string, bool) {
	for _, d := range domains {
		if NormalizeServer(d.Server) != wantNormalized {
			continue
		}
		t := strings.TrimSpace(d.Token)
		if t != "" {
			return t, true
		}
	}
	return "", false
}

func tokenFromCredentials(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		t := strings.TrimSpace(line)
		if t != "" {
			return t, true
		}
	}
	return "", false
}
