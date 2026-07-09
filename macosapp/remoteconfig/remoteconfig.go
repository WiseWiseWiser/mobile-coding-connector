// Package remoteconfig provides pure helpers for the remote macOS menu-bar app
// configuration (remote-agent-config.json), matching the CLI agentConfig schema.
package remoteconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConnectionState is the config-level (or post-probe) connection status.
type ConnectionState string

const (
	StateNotConfigured ConnectionState = "not_configured"
	StateNoDefault     ConnectionState = "no_default"
	StateOK            ConnectionState = "ok"
	StateUnauthorized  ConnectionState = "unauthorized"
	StateUnreachable   ConnectionState = "unreachable"
)

// Domain is a saved server+token pair.
type Domain struct {
	Server string `json:"server"`
	Token  string `json:"token,omitempty"`
}

// ProjectBinding maps (server, remote_dir) to a local git checkout path.
type ProjectBinding struct {
	Server    string `json:"server"`
	RemoteDir string `json:"remote_dir"`
	LocalPath string `json:"local_path"`
}

// Config is the persisted remote-agent configuration.
type Config struct {
	Default         string           `json:"default,omitempty"`
	Domains         []Domain         `json:"domains"`
	ProjectBindings []ProjectBinding `json:"project_bindings,omitempty"`
}

// ResolvedEndpoint is a normalized server + token from Resolve.
type ResolvedEndpoint struct {
	Server string
	Token  string
	OK     bool
}

// Load reads a remote-agent config JSON file.
// Missing file returns (nil, nil). Unreadable / other I/O or parse errors return error.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes cfg as indented JSON with mode 0600, creating parent directories.
func Save(path string, cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out := *cfg
	if out.Domains == nil {
		out.Domains = []Domain{}
	}
	data, err := json.MarshalIndent(&out, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

// NormalizeServer trims surrounding space and trailing slashes.
func NormalizeServer(server string) string {
	return strings.TrimRight(strings.TrimSpace(server), "/")
}

// Resolve picks a domain from cfg and returns a normalized endpoint + config-level state.
//
// Rules:
//   - cfg == nil or empty Domains → not_configured
//   - Default matches a domain (after normalize) → that domain, ok
//   - empty/unmatched Default + exactly one domain → that domain, ok
//   - empty/unmatched Default + multiple domains → no_default
func Resolve(cfg *Config) (ResolvedEndpoint, ConnectionState) {
	if cfg == nil || len(cfg.Domains) == 0 {
		return ResolvedEndpoint{}, StateNotConfigured
	}

	def := NormalizeServer(cfg.Default)
	if def != "" {
		for _, d := range cfg.Domains {
			if NormalizeServer(d.Server) == def {
				return resolvedFromDomain(d), StateOK
			}
		}
		// Unmatched default: fall through to single-domain or no_default.
	}

	if len(cfg.Domains) == 1 {
		return resolvedFromDomain(cfg.Domains[0]), StateOK
	}

	// Multi-domain with empty or unmatched default.
	return ResolvedEndpoint{}, StateNoDefault
}

func resolvedFromDomain(d Domain) ResolvedEndpoint {
	return ResolvedEndpoint{
		Server: NormalizeServer(d.Server),
		Token:  d.Token,
		OK:     true,
	}
}

// AuthorizationHeader formats a Bearer token header value.
// Empty token returns "" (caller should omit the header).
func AuthorizationHeader(token string) string {
	if token == "" {
		return ""
	}
	return "Bearer " + token
}

// FormatStatus returns exact user-facing status copy for a connection state.
func FormatStatus(state ConnectionState, server string) string {
	switch state {
	case StateNotConfigured:
		return "Not configured — open Configure… to add a remote server"
	case StateNoDefault:
		return "Multiple servers configured — open Configure… to pick a default"
	case StateOK:
		return "Connected to " + server
	case StateUnauthorized:
		return "Token rejected — open Configure… to update credentials"
	case StateUnreachable:
		return "Cannot reach " + server + " — retry or Test Connection"
	default:
		return ""
	}
}

// OpenBrowserURL returns the URL to open for a resolved remote endpoint.
// When OK and Server are set, returns Server as-is (token never included).
// Otherwise returns empty string.
func OpenBrowserURL(ep ResolvedEndpoint) string {
	if !ep.OK || ep.Server == "" {
		return ""
	}
	return ep.Server
}

// SelectDefaultDomain returns a copy of cfg with Default set to the matching
// domain's normalized server URL. serverURL is matched after NormalizeServer.
// Returns an error when cfg is nil or no domain matches.
func SelectDefaultDomain(cfg *Config, serverURL string) (*Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	norm := NormalizeServer(serverURL)
	if norm == "" {
		return nil, fmt.Errorf("server URL is empty")
	}
	var match Domain
	found := false
	for _, d := range cfg.Domains {
		if NormalizeServer(d.Server) == norm {
			match = d
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("domain not found: %s", serverURL)
	}

	out := *cfg
	if cfg.Domains != nil {
		out.Domains = make([]Domain, len(cfg.Domains))
		copy(out.Domains, cfg.Domains)
	}
	if cfg.ProjectBindings != nil {
		out.ProjectBindings = make([]ProjectBinding, len(cfg.ProjectBindings))
		copy(out.ProjectBindings, cfg.ProjectBindings)
	}
	out.Default = NormalizeServer(match.Server)
	return &out, nil
}

// DefaultConfigPath returns $home/.ai-critic/remote-agent-config.json — the same
// path used by remote-agent CLI and the remote menu-bar app.
func DefaultConfigPath(home string) string {
	return filepath.Join(home, ".ai-critic", "remote-agent-config.json")
}

// StatusFromFile is the shipped load→resolve→status path used by the remote
// menu-bar app refresh: read config at path, resolve endpoint, format status.
// Missing file yields not_configured status (empty server, resolved=false).
// Parse/I/O errors (other than missing) are returned as err.
func StatusFromFile(path string) (statusLine, server string, resolved bool, err error) {
	cfg, err := Load(path)
	if err != nil {
		return "", "", false, err
	}
	ep, state := Resolve(cfg)
	return FormatStatus(state, ep.Server), ep.Server, ep.OK, nil
}
