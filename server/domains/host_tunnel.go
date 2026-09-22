package domains

import (
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

func rotatePersistedTunnelName() (string, error) {
	persistTunnelMu.Lock()
	defer persistTunnelMu.Unlock()
	cfg, err := LoadDomains()
	if err != nil {
		return "", err
	}
	name := cloudflareSettings.HostPreferredTunnelNameUnique(cloudflareSettings.LocalHostname())
	cfg.TunnelName = name
	if err := SaveDomains(cfg); err != nil {
		return "", err
	}
	return name, nil
}

// StartHostDomainTunnel starts a host Cloudflare tunnel using the persisted
// ai-critic-* tunnel name. On missing local credentials it allocates a unique
// name and retries once.
func StartHostDomainTunnel(domain string, port int, logFn cloudflareSettings.LogFunc) (*cloudflareSettings.DomainTunnelStatus, error) {
	if serverqemu.Enabled() {
		return cloudflareSettings.StartDomainTunnel(domain, port, "", logFn)
	}
	name, err := EnsurePersistedTunnelName()
	if err != nil {
		return nil, err
	}
	if logFn == nil {
		logFn = func(string) {}
	}
	logFn("using tunnel name " + name)
	status, err := cloudflareSettings.StartDomainTunnel(domain, port, name, logFn)
	if err == nil {
		return status, nil
	}
	if !isMissingTunnelCreds(err) {
		return nil, err
	}
	logFn(fmt.Sprintf("tunnel %s has no local credentials; allocating a unique name", name))
	name, err = rotatePersistedTunnelName()
	if err != nil {
		return nil, err
	}
	logFn("retry with tunnel name " + name)
	return cloudflareSettings.StartDomainTunnel(domain, port, name, logFn)
}
