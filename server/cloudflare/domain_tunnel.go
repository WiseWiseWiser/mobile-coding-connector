package cloudflare

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
)

// DomainTunnelStatus describes the runtime status of a domain tunnel.
type DomainTunnelStatus struct {
	Status    string `json:"status"`               // "stopped", "connecting", "active", "error"
	TunnelURL string `json:"tunnel_url,omitempty"` // the public URL (https://<domain>)
	Error     string `json:"error,omitempty"`
}

// ParseBaseDomain extracts the base domain from a full hostname.
// e.g. "sub.example.com" -> "example.com", "a.b.example.co.uk" -> "example.co.uk"
// Simple approach: take the last two parts separated by dots.
func ParseBaseDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) <= 2 {
		return domain
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

// GetDomainTunnelStatus returns the current status of a domain tunnel.
// With the tunnel group, this checks if the domain is registered as a mapping in the core group.
func GetDomainTunnelStatus(domain string) DomainTunnelStatus {
	tg := GetTunnelGroupManager().GetCoreGroup()
	if tg == nil {
		return DomainTunnelStatus{Status: "stopped"}
	}

	// Check if this domain is in the core tunnel group mappings
	mappings := tg.ListMappings()
	for _, m := range mappings {
		if m.Hostname == domain {
			if tg.IsRunning() {
				return DomainTunnelStatus{
					Status:    "active",
					TunnelURL: fmt.Sprintf("https://%s", domain),
				}
			}
			return DomainTunnelStatus{
				Status:    "connecting",
				TunnelURL: fmt.Sprintf("https://%s", domain),
			}
		}
	}

	return DomainTunnelStatus{Status: "stopped"}
}

// LogFunc is a callback for streaming log messages during tunnel operations.
type LogFunc func(message string)

// EnsureUnifiedTunnelConfigured ensures the unified tunnel manager has a tunnel configured.
// It checks if a tunnel is already configured and reuses it.
// If not, it creates a new tunnel using the provided name.
// If tunnelName is empty, it will either:
// - Find an existing tunnel to reuse
// - Create a tunnel named after an owned domain, or "ai-critic-default" as last resort
// Returns the tunnel reference, tunnel ID, and credentials file path.
func EnsureUnifiedTunnelConfigured(tunnelName string, logFn LogFunc) (tunnelRef string, tunnelID string, credFile string, err error) {
	if logFn == nil {
		logFn = func(string) {}
	}

	utm := GetUnifiedTunnelManager()

	// Check if unified tunnel manager is already configured
	existingConfig := utm.GetConfig()
	if existingConfig != nil {
		// Reuse existing tunnel configuration
		tunnelRef = existingConfig.TunnelName
		tunnelID = existingConfig.TunnelID
		credFile = existingConfig.CredentialsFile
		logFn(fmt.Sprintf("Reusing existing unified tunnel: %s (id=%s)", tunnelRef, tunnelID))
		return tunnelRef, tunnelID, credFile, nil
	}

	// No existing config, need to create/configure a tunnel
	// If tunnelName is not provided, derive one from owned domains or use default
	if tunnelName == "" {
		// Try to use first owned domain as tunnel name
		ownedDomains := GetOwnedDomains()
		if len(ownedDomains) > 0 {
			tunnelName = DefaultTunnelName(ownedDomains[0])
			logFn(fmt.Sprintf("Using owned domain as tunnel name: %s", tunnelName))
		} else {
			// Generate default tunnel name: ai-critic-default-<hostname>-<hostip>
			hostname := "unknown-host"
			if h, err := os.Hostname(); err == nil {
				hostname = h
			}

			hostIP := "unknown-ip"
			// Try to get the primary IP address
			if addrs, err := net.InterfaceAddrs(); err == nil {
				for _, addr := range addrs {
					if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
						if ipnet.IP.To4() != nil {
							hostIP = ipnet.IP.String()
							break
						}
					}
				}
			}

			// Sanitize hostname and IP for tunnel name (only alphanumeric and hyphens)
			hostname = strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
					return r
				}
				return '-'
			}, hostname)
			hostIP = strings.Map(func(r rune) rune {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
					return r
				}
				return '-'
			}, hostIP)

			tunnelName = fmt.Sprintf("ai-critic-default-%s-%s", hostname, hostIP)

			// Truncate tunnel name if too long (Cloudflare limit is ~32 chars or so)
			// Keep the unique parts: hostname up to 15 chars, then IP
			if len(tunnelName) > 40 {
				// Extract just the essential parts
				shortHostname := hostname
				if len(hostname) > 15 {
					shortHostname = hostname[:15]
				}
				tunnelName = fmt.Sprintf("ai-critic-%s-%s", shortHostname, hostIP)
			}

			logFn(fmt.Sprintf("No owned domains, using default tunnel name: %s", tunnelName))
		}
	}

	logFn("No existing tunnel configured, creating new tunnel '" + tunnelName + "'...")
	tunnelRef, err = FindOrCreateTunnel(tunnelName)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to find/create tunnel: %v", err)
	}
	logFn("Created/found tunnel: " + tunnelRef)

	// Get tunnel credentials
	tunnelID, credFile, err = EnsureTunnelExists(tunnelRef)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get tunnel credentials: %v", err)
	}
	logFn(fmt.Sprintf("Got tunnel credentials: id=%s", tunnelID))

	// Configure the unified tunnel manager with this tunnel's config
	utm.SetConfig(config.CloudflareTunnelConfig{
		TunnelName:      tunnelRef,
		TunnelID:        tunnelID,
		CredentialsFile: credFile,
	})
	logFn("Unified tunnel manager configured with: " + tunnelRef)

	return tunnelRef, tunnelID, credFile, nil
}

// EnsureGroupTunnelConfigured ensures the tunnel group has a tunnel configured.
// It checks if a tunnel is already configured and reuses it.
// If not, it creates a new tunnel using the provided name.
// If tunnelName is empty, it generates a tunnel name using the group name.
// Returns the tunnel reference, tunnel ID, and credentials file path.
func EnsureGroupTunnelConfigured(group string, tunnelName string, logFn LogFunc) (tunnelRef string, tunnelID string, credFile string, err error) {
	if logFn == nil {
		logFn = func(string) {}
	}

	tg := GetTunnelGroupManager().GetGroup(group)
	if tg == nil {
		return "", "", "", fmt.Errorf("unknown tunnel group: %s", group)
	}

	utm := tg.tunnelMgr

	existingConfig := utm.GetConfig()
	if existingConfig != nil {
		tunnelRef = existingConfig.TunnelName
		tunnelID = existingConfig.TunnelID
		credFile = existingConfig.CredentialsFile
		logFn(fmt.Sprintf("Reusing existing tunnel for group %s: %s (id=%s)", group, tunnelRef, tunnelID))
		return tunnelRef, tunnelID, credFile, nil
	}

	if tunnelName == "" {
		tunnelName = GenerateTunnelName(group)
		logFn(fmt.Sprintf("Generated tunnel name for group %s: %s", group, tunnelName))
	}

	logFn(fmt.Sprintf("No existing tunnel configured for group %s, creating new tunnel '%s'...", group, tunnelName))
	tunnelRef, err = FindOrCreateTunnel(tunnelName)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to find/create tunnel: %v", err)
	}
	logFn("Created/found tunnel: " + tunnelRef)

	tunnelID, credFile, err = EnsureTunnelExists(tunnelRef)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get tunnel credentials: %v", err)
	}
	logFn(fmt.Sprintf("Got tunnel credentials: id=%s", tunnelID))

	utm.SetConfig(config.CloudflareTunnelConfig{
		TunnelName:      tunnelRef,
		TunnelID:        tunnelID,
		CredentialsFile: credFile,
	})
	logFn(fmt.Sprintf("Tunnel group %s configured with: %s", group, tunnelRef))

	return tunnelRef, tunnelID, credFile, nil
}

// StartDomainTunnel starts a cloudflare named tunnel for the given domain.
// The domain is routed via DNS to the tunnel, and an ingress rule maps it
// to http://localhost:<port>.
// tunnelName is the cloudflare tunnel name to use if no tunnel is already configured.
// logFn is an optional callback for streaming log messages.
//
// This function uses the core tunnel group, so multiple domains share
// a single cloudflared process. If a tunnel is already configured, it reuses it.
func StartDomainTunnel(domain string, port int, tunnelName string, logFn LogFunc) (*DomainTunnelStatus, error) {
	if logFn == nil {
		logFn = func(string) {}
	}

	// Check if this domain is already running
	status := GetDomainTunnelStatus(domain)
	if status.Status == "active" {
		logFn("Tunnel already running for " + domain)
		return &status, nil
	}

	// Ensure core tunnel group has a tunnel configured (reuses existing or creates new)
	tunnelRef, _, _, err := EnsureGroupTunnelConfigured(GroupCore, tunnelName, logFn)
	if err != nil {
		return nil, err
	}

	// Create DNS route
	logFn("Creating DNS route for " + domain + "...")
	if err := CreateDNSRoute(tunnelRef, domain); err != nil {
		return nil, err
	}
	logFn("DNS route created")

	// Add ingress rule to core tunnel group
	tg := GetTunnelGroupManager().GetCoreGroup()
	localURL := fmt.Sprintf("http://localhost:%d", port)
	mappingID := fmt.Sprintf("domain-%s", domain)
	mapping := &IngressMapping{
		ID:       mappingID,
		Hostname: domain,
		Service:  localURL,
		Source:   fmt.Sprintf("domain:%s", domain),
	}

	logFn(fmt.Sprintf("Adding ingress rule: %s -> %s", domain, localURL))
	if err := tg.AddMapping(mapping); err != nil {
		return nil, fmt.Errorf("failed to add mapping to core tunnel: %v", err)
	}
	logFn("Ingress rule added, tunnel started")

	return &DomainTunnelStatus{
		Status:    "active",
		TunnelURL: fmt.Sprintf("https://%s", domain),
	}, nil
}

// StopDomainTunnel stops the tunnel for the given domain.
// tunnelName is the cloudflare tunnel name; if empty, a default is derived from the domain.
func StopDomainTunnel(domain string, tunnelName string) error {
	_ = tunnelName // not used with tunnel group, but kept for API compatibility

	tg := GetTunnelGroupManager().GetCoreGroup()

	// Remove the mapping from core tunnel group
	mappingID := fmt.Sprintf("domain-%s", domain)
	if err := tg.RemoveMapping(mappingID); err != nil {
		return fmt.Errorf("failed to remove mapping: %v", err)
	}

	// If no more mappings, the tunnel group will automatically stop the process
	return nil
}
