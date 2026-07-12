package unified_tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/xhd2015/ai-critic/server/cmdjson"
	"github.com/xhd2015/dot-pkgs/go-pkgs/cloudflare"
)

// TunnelInfo represents a Cloudflare tunnel.
type TunnelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CreatedAt   string `json:"created_at,omitempty"`
	Connections []any  `json:"connections,omitempty"`
}

// CloudflaredConfig represents a cloudflared config.yml structure.
type CloudflaredConfig struct {
	Tunnel          string        `yaml:"tunnel"`
	CredentialsFile string        `yaml:"credentials-file"`
	Ingress         []IngressRule `yaml:"ingress"`
}

// IngressRule represents a single cloudflared ingress entry.
type IngressRule struct {
	Hostname string `yaml:"hostname,omitempty"`
	Service  string `yaml:"service"`
}

// IsUUID checks if a string looks like a UUID (8-4-4-4-12 hex format).
// Delegates to the shared cloudflare package.
func IsUUID(s string) bool {
	return cloudflare.IsUUID(s)
}

// FindTunnelIDAndCreds resolves the tunnel ID and credentials file for the given tunnel reference (name or ID).
func FindTunnelIDAndCreds(tunnelRef string) (tunnelID string, credFile string, err error) {
	infoOut, err := exec.Command("cloudflared", "tunnel", "info", tunnelRef).CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("tunnel %q not found: %s", tunnelRef, strings.TrimSpace(string(infoOut)))
	}

	infoStr := string(infoOut)

	// Parse tunnel ID (UUID) from the info output
	for _, line := range strings.Split(infoStr, "\n") {
		for _, part := range strings.Fields(line) {
			if IsUUID(part) {
				tunnelID = part
				break
			}
		}
		if tunnelID != "" {
			break
		}
	}

	if tunnelID == "" {
		return "", "", fmt.Errorf("could not determine tunnel ID for %q", tunnelRef)
	}

	// Look for credentials file in default location
	homeDir, _ := os.UserHomeDir()
	credFile = filepath.Join(homeDir, ".cloudflared", tunnelID+".json")
	if _, statErr := os.Stat(credFile); statErr != nil {
		return "", "", fmt.Errorf("credentials file not found: %s", credFile)
	}

	return tunnelID, credFile, nil
}

// EnsureTunnelExists checks if the named tunnel exists. If not, it creates one.
// Returns the tunnel ID and credentials file path.
func EnsureTunnelExists(tunnelRef string) (tunnelID string, credFile string, err error) {
	// First try to get info about the existing tunnel
	tunnelID, credFile, err = FindTunnelIDAndCreds(tunnelRef)
	if err == nil {
		return tunnelID, credFile, nil
	}

	// Tunnel doesn't exist — create it
	createOut, createErr := exec.Command("cloudflared", "tunnel", "create", tunnelRef).CombinedOutput()
	createStr := strings.TrimSpace(string(createOut))
	if createErr != nil {
		return "", "", fmt.Errorf("failed to create tunnel %q: %s", tunnelRef, createStr)
	}

	// Parse tunnel ID from: "Created tunnel <name> with id <uuid>"
	idRegex := regexp.MustCompile(`with id ([a-f0-9-]+)`)
	if match := idRegex.FindStringSubmatch(createStr); len(match) > 1 {
		tunnelID = match[1]
	}

	// Parse credentials file from: "...credentials written to <path>.json"
	credRegex := regexp.MustCompile(`credentials written to (.+\.json)`)
	if match := credRegex.FindStringSubmatch(createStr); len(match) > 1 {
		credFile = match[1]
	}

	// Fall back to default location
	if credFile == "" && tunnelID != "" {
		homeDir, _ := os.UserHomeDir()
		credFile = filepath.Join(homeDir, ".cloudflared", tunnelID+".json")
		if _, statErr := os.Stat(credFile); statErr != nil {
			credFile = ""
		}
	}

	if tunnelID == "" {
		return "", "", fmt.Errorf("could not determine tunnel ID for %q", tunnelRef)
	}
	if credFile == "" {
		return "", "", fmt.Errorf("could not find credentials file for tunnel %q (id: %s)", tunnelRef, tunnelID)
	}

	return tunnelID, credFile, nil
}

// FindOrCreateTunnel finds an existing tunnel (preferring the given name) or creates one.
// Returns the tunnel name to use.
func FindOrCreateTunnel(preferredName string) (string, error) {
	result, err := cmdjson.Run[[]TunnelInfo](exec.Command("cloudflared", "tunnel", "list", "--output", "json"))
	if err == nil {
		if warning := result.Warning(); warning != "" {
			fmt.Fprintf(os.Stderr, "[cloudflare] cloudflared tunnel list warning: %s\n", warning)
		}
		// Look for a tunnel with the preferred name.
		for _, t := range result.Data {
			if t.Name == preferredName {
				return t.Name, nil
			}
		}
	}

	// Tunnel with preferred name not found — create one
	createOut, err := exec.Command("cloudflared", "tunnel", "create", preferredName).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create tunnel %q: %s", preferredName, strings.TrimSpace(string(createOut)))
	}

	return preferredName, nil
}

// CreateDNSRoute creates a DNS route pointing the hostname to the tunnel.
// Ignores "already exists" errors.
func CreateDNSRoute(tunnelRef, hostname string) error {
	out, err := exec.Command("cloudflared", "tunnel", "route", "dns", "--overwrite-dns", tunnelRef, hostname).CombinedOutput()
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		if !strings.Contains(outStr, "already exists") && !strings.Contains(outStr, "Added CNAME") {
			return fmt.Errorf("failed to create DNS route: %s", outStr)
		}
	}
	return nil
}

// WriteCloudflaredConfig writes a cloudflared config YAML file.
// Delegates to the shared cloudflare.WriteConfig.
func WriteCloudflaredConfig(path string, cfg *CloudflaredConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	shared := &cloudflare.Config{
		Tunnel:          cfg.Tunnel,
		CredentialsFile: cfg.CredentialsFile,
	}
	for _, rule := range cfg.Ingress {
		shared.Ingress = append(shared.Ingress, cloudflare.IngressRule{
			Hostname: rule.Hostname,
			Service:  rule.Service,
		})
	}
	return cloudflare.WriteConfig(path, shared)
}

// DefaultConfigDir returns the default cloudflared config directory (~/.cloudflared).
func DefaultConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}
	return filepath.Join(homeDir, ".cloudflared"), nil
}
