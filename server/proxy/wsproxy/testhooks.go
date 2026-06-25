package wsproxy

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// SetTestConfigDir redirects ws-proxy config reads/writes for tests.
func SetTestConfigDir(dir string) {
	_testConfigDir = dir
	if dir == "" {
		clearTestTunnelMapped()
	}
}

// SaveTestConfig persists cfg using the active test config directory.
func SaveTestConfig(cfg *Config) error {
	return SaveConfig(cfg)
}

// NewTestManager constructs a Manager with preset public URL state.
func NewTestManager(publicURL string, isTmp bool) *Manager {
	return &Manager{publicURL: publicURL, isTmp: isTmp}
}

// IsXrayAliveForTest exposes local xray health probing for tests.
func IsXrayAliveForTest(port int, wsPath string) bool {
	return isXrayAlive(port, wsPath)
}

// HostFromPublicURL strips the scheme from a public URL.
func HostFromPublicURL(publicURL string) string {
	host := strings.TrimPrefix(publicURL, "https://")
	return strings.TrimPrefix(host, "http://")
}

// ExtractPortFromURL parses the port from an http(s) URL or host:port string.
func ExtractPortFromURL(raw string) int {
	raw = strings.TrimPrefix(raw, "http://")
	raw = strings.TrimPrefix(raw, "https://")
	_, portStr, err := net.SplitHostPort(raw)
	if err != nil {
		return 80
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 80
	}
	return port
}

// DerivePublicURL rebuilds the public URL from persisted ws-proxy config and base domain.
func DerivePublicURL(cfg *Config, baseDomain string) string {
	if cfg == nil || baseDomain == "" {
		return ""
	}
	instanceID := resolveInstanceID(cfg)
	hostname := fmt.Sprintf("%s-%s.%s", cfg.Subdomain, instanceID, baseDomain)
	return fmt.Sprintf("https://%s", hostname)
}

// IsClientReady reports whether ws-proxy is usable by VMess clients.
func IsClientReady(status *Status, tunnelMapped bool, vmessLink string) bool {
	if status == nil {
		return false
	}
	return status.Running && status.PublicURL != "" && tunnelMapped && vmessLink != ""
}

// RecoverTunnel attempts to restore a missing Cloudflare ingress mapping.
func RecoverTunnel(m *Manager) error {
	return m.Recover()
}