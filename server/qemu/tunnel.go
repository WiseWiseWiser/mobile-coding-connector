package qemu

import (
	"fmt"
	"sync"
)

var (
	activeMu      sync.Mutex
	activeDomains = map[string]string{} // domain -> tunnel name
)

// MarkDomainActive records that domain is served by guest cloudflared.
func MarkDomainActive(domain, tunnelName string) {
	activeMu.Lock()
	defer activeMu.Unlock()
	activeDomains[domain] = tunnelName
}

// MarkDomainInactive clears guest-tunnel tracking for domain.
func MarkDomainInactive(domain string) {
	activeMu.Lock()
	defer activeMu.Unlock()
	delete(activeDomains, domain)
}

// IsDomainActive reports whether domain was started via guest cloudflared.
func IsDomainActive(domain string) bool {
	activeMu.Lock()
	defer activeMu.Unlock()
	_, ok := activeDomains[domain]
	return ok
}

func domainsForTunnel(tunnelName string) []string {
	activeMu.Lock()
	defer activeMu.Unlock()
	var hosts []string
	for d, n := range activeDomains {
		if n == tunnelName {
			hosts = append(hosts, d)
		}
	}
	return hosts
}

// StartDomainTunnelInGuest ensures qemu is up and starts a named tunnel in the
// guest with origin http://10.0.2.2:<port> for the given domain.
func StartDomainTunnelInGuest(domain string, port int, tunnelName string, logFn func(string)) (tunnelURL string, err error) {
	if logFn == nil {
		logFn = func(string) {}
	}
	if tunnelName == "" {
		tunnelName = "ai-critic-core"
	}
	origin := fmt.Sprintf("http://10.0.2.2:%d", port)

	m := DefaultManager()
	logFn("ensuring qemu guest is running...")
	kv, err := m.EnsureRunning()
	if err != nil {
		return "", fmt.Errorf("ensure qemu: %w", err)
	}
	if kv["qemu_alive"] != "yes" && kv["skip"] != "running" {
		// EnsureRunning may return start KV without qemu_alive key when pid set
		if kv["qemu_pid"] == "" {
			return "", fmt.Errorf("qemu guest not running")
		}
	}

	// Merge with already-active hostnames so AutoStart of multiple domains
	// does not drop earlier ingress rules (named-start rewrites the full config).
	hosts := domainsForTunnel(tunnelName)
	found := false
	for _, h := range hosts {
		if h == domain {
			found = true
			break
		}
	}
	if !found {
		hosts = append(hosts, domain)
	}

	logFn(fmt.Sprintf("starting guest named tunnel %q origin=%s hostnames=%v", tunnelName, origin, hosts))
	kv, err = m.CFNamedStart(tunnelName, origin, hosts)
	if err != nil {
		return "", fmt.Errorf("guest cloudflared named-start: %w", err)
	}
	if kv["cf_alive"] != "yes" && kv["named"] != "yes" {
		// named-start prints named=yes on success; tolerate cf_alive from status keys
		if kv["qemu_alive"] == "no" {
			return "", fmt.Errorf("guest cloudflared named-start failed: %#v", kv)
		}
	}
	MarkDomainActive(domain, tunnelName)
	return "https://" + domain, nil
}

// StopDomainTunnelInGuest removes domain from guest tracking. If other domains
// still use the same tunnel, re-starts named tunnel with the remaining hosts;
// otherwise stops guest cloudflared.
func StopDomainTunnelInGuest(domain string, port int, tunnelName string, logFn func(string)) error {
	if logFn == nil {
		logFn = func(string) {}
	}
	activeMu.Lock()
	if tunnelName == "" {
		tunnelName = activeDomains[domain]
	}
	delete(activeDomains, domain)
	var remaining []string
	for d, n := range activeDomains {
		if n == tunnelName {
			remaining = append(remaining, d)
		}
	}
	activeMu.Unlock()

	if tunnelName == "" || len(remaining) == 0 {
		logFn("stopping guest cloudflared (no remaining domains)...")
		return DefaultManager().CFStop()
	}
	if port <= 0 {
		port = 23712
	}
	origin := GuestOriginForPort(port)
	logFn(fmt.Sprintf("rebuilding guest named tunnel %q for %v", tunnelName, remaining))
	_, err := DefaultManager().CFNamedStart(tunnelName, origin, remaining)
	return err
}

// GuestOriginForPort returns the SLIRP host-gateway origin for a host port.
func GuestOriginForPort(port int) string {
	return fmt.Sprintf("http://10.0.2.2:%d", port)
}
