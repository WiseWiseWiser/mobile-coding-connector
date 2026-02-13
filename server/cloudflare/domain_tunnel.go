package cloudflare

import (
	"fmt"
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
// With the unified tunnel manager, this checks if the domain is registered as a mapping.
func GetDomainTunnelStatus(domain string) DomainTunnelStatus {
	utm := GetUnifiedTunnelManager()

	// Check if this domain is in the unified tunnel mappings
	mappings := utm.ListMappings()
	for _, m := range mappings {
		if m.Hostname == domain {
			if utm.IsRunning() {
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

// StartDomainTunnel starts a cloudflare named tunnel for the given domain.
// The domain is routed via DNS to the tunnel, and an ingress rule maps it
// to http://localhost:<port>.
// tunnelName is the cloudflare tunnel name to use; if empty, a default is derived from the domain.
// logFn is an optional callback for streaming log messages.
//
// This function uses the unified tunnel manager, so multiple domains share
// a single cloudflared process.
func StartDomainTunnel(domain string, port int, tunnelName string, logFn LogFunc) (*DomainTunnelStatus, error) {
	if logFn == nil {
		logFn = func(string) {}
	}

	utm := GetUnifiedTunnelManager()

	// Check if this domain is already running
	status := GetDomainTunnelStatus(domain)
	if status.Status == "active" {
		logFn("Tunnel already running for " + domain)
		return &status, nil
	}

	if tunnelName == "" {
		tunnelName = DefaultTunnelName(domain)
	}

	// Find or create a tunnel
	logFn("Finding or creating tunnel '" + tunnelName + "'...")
	tunnelRef, err := FindOrCreateTunnel(tunnelName)
	if err != nil {
		return nil, fmt.Errorf("failed to find/create tunnel: %v", err)
	}
	logFn("Using tunnel: " + tunnelRef)

	// Get tunnel ID and credentials
	tunnelID, credFile, err := EnsureTunnelExists(tunnelRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get tunnel credentials: %v", err)
	}
	logFn(fmt.Sprintf("Got tunnel credentials: id=%s", tunnelID))

	// Configure the unified tunnel manager with this tunnel's config
	utm.SetConfig(config.CloudflareTunnelConfig{
		TunnelName:      tunnelRef,
		TunnelID:        tunnelID,
		CredentialsFile: credFile,
	})

	// Create DNS route
	logFn("Creating DNS route for " + domain + "...")
	if err := CreateDNSRoute(tunnelRef, domain); err != nil {
		return nil, err
	}
	logFn("DNS route created")

	// Add ingress rule to unified tunnel manager
	localURL := fmt.Sprintf("http://localhost:%d", port)
	mappingID := fmt.Sprintf("domain-%s", domain)
	mapping := &IngressMapping{
		ID:       mappingID,
		Hostname: domain,
		Service:  localURL,
		Source:   fmt.Sprintf("domain:%s", domain),
	}

	logFn(fmt.Sprintf("Adding ingress rule: %s -> %s", domain, localURL))
	if err := utm.AddMapping(mapping); err != nil {
		return nil, fmt.Errorf("failed to add mapping to unified tunnel: %v", err)
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
	_ = tunnelName // not used with unified tunnel manager, but kept for API compatibility

	utm := GetUnifiedTunnelManager()

	// Remove the mapping from unified tunnel manager
	mappingID := fmt.Sprintf("domain-%s", domain)
	if err := utm.RemoveMapping(mappingID); err != nil {
		return fmt.Errorf("failed to remove mapping: %v", err)
	}

	// If no more mappings, the unified tunnel manager will automatically stop the process
	return nil
}
