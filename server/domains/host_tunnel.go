package domains

import (
	"context"
	"fmt"
	"strings"
	"sync"

	cloudflareSettings "github.com/xhd2015/ai-critic/server/cloudflare"
	serverqemu "github.com/xhd2015/ai-critic/server/qemu"
)

var persistTunnelMu sync.Mutex

func isMissingTunnelCreds(err error) bool {
	return err != nil && strings.Contains(err.Error(), "credentials file not found")
}

// EnsurePersistedTunnelName returns server-domains.json tunnel_name, generating
// ai-critic-<hostname> once when empty.
func EnsurePersistedTunnelName() (string, error) {
	persistTunnelMu.Lock()
	defer persistTunnelMu.Unlock()
	cfg, err := LoadDomains()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(cfg.TunnelName) != "" {
		return cfg.TunnelName, nil
	}
	name := cloudflareSettings.HostPreferredTunnelName(cloudflareSettings.LocalHostname())
	cfg.TunnelName = name
	if err := SaveDomains(cfg); err != nil {
		return "", err
	}
	return name, nil
}

// StartHostDomainTunnel uses the persisted shared tunnel identity. Missing
// credentials must be repaired rather than silently rotating sibling routes.
func StartHostDomainTunnel(domain string, port int, logFn cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
	return startHostDomainTunnel(domain, port, logFn, false)
}

// ForceStartHostDomainTunnel is StartHostDomainTunnel that does not keep a
// tunnel already marked active. Used when switching backends.
func ForceStartHostDomainTunnel(domain string, port int, logFn cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
	return startHostDomainTunnel(domain, port, logFn, true)
}

// Both bootstrap and owned port-forward retries share this backend and must
// preserve the persisted tunnel identity, even if credentials are missing.
func startBootstrapDomainTunnel(domain string, port int, logFn cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
	return startHostDomainTunnel(domain, port, logFn, false)
}

func retryBootstrapDomainTunnel(domain string, port int, logFn cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
	return startHostDomainTunnel(domain, port, logFn, true)
}

func startHostDomainTunnel(domain string, port int, logFn cloudflareSettings.LogFunc, force bool) (*cloudflareSettings.DomainTunnelStatus, error) {
	start := cloudflareSettings.StartDomainTunnel
	if force {
		start = cloudflareSettings.ForceStartDomainTunnel
	}
	if serverqemu.Enabled() {
		return start(domain, port, "", logFn)
	}
	name, err := EnsurePersistedTunnelName()
	if err != nil {
		return nil, err
	}
	if logFn == nil {
		logFn = func(string) {}
	}
	logFn("using tunnel name " + name)
	status, err := start(domain, port, name, logFn)
	if err == nil {
		return status, nil
	}
	if isMissingTunnelCreds(err) {
		return nil, fmt.Errorf("%w; restore credentials for the configured tunnel without replacing it", err)
	}
	if strings.Contains(err.Error(), "could not determine tunnel ID") {
		return nil, fmt.Errorf("%w; check DNS/network access to Cloudflare and tunnel credentials", err)
	}
	return nil, err
}

// StopHostDomainTunnel tears down the tunnel StartHostDomainTunnel started.
//
// In proxy mode the publish owns a mapping on the edge, and only stopping its
// session deletes that mapping. Removing the cloudflared backend route instead
// (as the owned-domain port forward used to) strands the hostname: the edge
// keeps serving a mapping whose dial pool is dead, and the next start sees a
// live-looking pool, skips publishing, and leaves the hostname broken until the
// server restarts.
func StopHostDomainTunnel(domain string, logFn cloudflareSettings.LogFunc) error {
	if logFn == nil {
		logFn = func(string) {}
	}
	if cloudflareSettings.ProxyModeEnabled() {
		logFn("stopping proxy publish for " + domain)
		return cloudflareSettings.StopDomainTunnel(domain, "")
	}
	b := serverqemu.CloudflaredBackend()
	if serverqemu.Enabled() || serverqemu.IsDomainActive(domain) {
		return cloudflareSettings.StopDomainTunnel(domain, "")
	}
	cfg, err := LoadDomains()
	if err != nil {
		return fmt.Errorf("load tunnel name before stopping: %w", err)
	}
	name := strings.TrimSpace(cfg.TunnelName)
	if name == "" {
		return fmt.Errorf("cannot stop managed tunnel without its persisted name")
	}
	return b.RemoveRoute(context.Background(), name, domain)
}
