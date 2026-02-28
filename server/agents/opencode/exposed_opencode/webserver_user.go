package exposed_opencode

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/proxy/basic_auth_proxy"
	"github.com/xhd2015/lifelog-private/ai-critic/server/proxy/portforward"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

// GetWebServerStatus checks if the OpenCode web server is running and if its port is mapped.
func GetWebServerStatus() (*WebServerStatus, error) {
	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}

	proxyPort := settings.WebServer.Port
	status := &WebServerStatus{
		Port:             proxyPort,
		Domain:           settings.DefaultDomain,
		TargetPreference: settings.WebServer.TargetPreference,
		ExposedDomain:    settings.WebServer.ExposedDomain,
		OpencodePort:     0,
	}

	// Check if auth proxy binary exists and get its full path.
	if path, err := tool_resolve.LookPath("basic-auth-proxy"); err == nil {
		status.AuthProxyFound = true
		status.AuthProxyPath = path
	} else {
		status.AuthProxyFound = false
		status.AuthProxyPath = ""
	}

	// Check if auth proxy is running on the proxy port.
	if settings.WebServer.AuthProxyEnabled {
		status.AuthProxyRunning = IsAuthProxyRunning(proxyPort)
		if status.AuthProxyRunning {
			status.OpencodePort = GetOpencodeInternalPort(proxyPort)
		}
		status.Running = status.AuthProxyRunning
	} else {
		status.Running = IsWebServerRunning(proxyPort)
		status.OpencodePort = proxyPort
	}

	if status.Running && settings.DefaultDomain != "" {
		status.PortMapped = isPortMappedToDomain(proxyPort, settings.DefaultDomain)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		status.ConfigPath = filepath.Join(home, ".local", "share", "opencode", "config.json")
	}

	return status, nil
}

// IsAuthProxyRunning checks if the auth proxy is running on the given port.
func IsAuthProxyRunning(port int) bool {
	return basic_auth_proxy.IsRunning(port)
}

// GetOpencodeInternalPort returns the internal opencode port from the proxy config file.
func GetOpencodeInternalPort(_ int) int {
	return basic_auth_proxy.GetBackendPort()
}

// IsWebServerRunning checks if the OpenCode web server is running on the given port.
func IsWebServerRunning(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func isPortMappedToDomain(port int, domain string) bool {
	url := fmt.Sprintf("http://%s/global/health", domain)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// IsWebServerEnabled checks if the web server is enabled in settings.
func IsWebServerEnabled() bool {
	settings, err := LoadSettings()
	if err != nil {
		return false
	}
	return settings.WebServer.Enabled
}

// GetWebServerPort returns the configured web server port.
func GetWebServerPort() int {
	settings, err := LoadSettings()
	if err != nil {
		return 4096
	}
	if settings.WebServer.Port == 0 {
		return 4096
	}
	return settings.WebServer.Port
}

// CheckPortMappingStatus returns a human-readable status of the port mapping.
func CheckPortMappingStatus(port int, domain string) string {
	if domain == "" {
		return "No domain configured"
	}
	if !IsWebServerRunning(port) {
		return "Web server is not running"
	}
	if isPortMappedToDomain(port, domain) {
		return fmt.Sprintf("Port %d is successfully mapped to %s", port, domain)
	}
	return fmt.Sprintf("Port %d is not mapped to %s. Check your DNS or tunnel configuration.", port, domain)
}

// ExtractDomainFromURL extracts the domain from a URL string.
func ExtractDomainFromURL(urlStr string) string {
	urlStr = strings.TrimSpace(urlStr)
	urlStr = strings.TrimPrefix(urlStr, "http://")
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "ws://")
	urlStr = strings.TrimPrefix(urlStr, "wss://")

	if idx := strings.Index(urlStr, "/"); idx != -1 {
		urlStr = urlStr[:idx]
	}
	if idx := strings.Index(urlStr, "?"); idx != -1 {
		urlStr = urlStr[:idx]
	}
	if idx := strings.Index(urlStr, ":"); idx != -1 {
		urlStr = urlStr[:idx]
	}
	return urlStr
}

func getBaseDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return ""
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return domain
	}

	if len(parts) >= 3 && len(parts[len(parts)-1]) == 2 && len(parts[len(parts)-2]) <= 3 {
		return strings.Join(parts[len(parts)-3:], ".")
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

// DomainMatchesOwned checks if the given domain's base domain matches any owned domain.
func DomainMatchesOwned(domain string) (bool, string) {
	if domain == "" {
		return false, ""
	}

	ownedDomains := cloudflare.GetOwnedDomains()
	if len(ownedDomains) == 0 {
		return false, ""
	}

	baseDomain := getBaseDomain(domain)
	for _, owned := range ownedDomains {
		if strings.EqualFold(baseDomain, owned) {
			return true, owned
		}
	}
	return false, ""
}

// MapDomainViaCloudflare maps the web server port to the configured domain using Cloudflare.
func MapDomainViaCloudflare(provider string) (*MapDomainResponse, error) {
	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}

	if settings.DefaultDomain == "" {
		return &MapDomainResponse{
			Success: false,
			Message: "No default domain configured",
		}, nil
	}

	if !IsWebServerRunning(settings.WebServer.Port) {
		return &MapDomainResponse{
			Success: false,
			Message: "Web server is not running",
		}, nil
	}

	matches, _ := DomainMatchesOwned(settings.DefaultDomain)
	if !matches {
		return &MapDomainResponse{
			Success: false,
			Message: fmt.Sprintf("Domain %s does not match any owned domain (base: %s)", settings.DefaultDomain, getBaseDomain(settings.DefaultDomain)),
		}, nil
	}

	if provider == "" {
		provider = portforward.ProviderCloudflareOwned
	}

	label := settings.DefaultDomain
	pf, err := portforward.GetDefaultManager().Add(settings.WebServer.Port, label, provider)
	if err != nil {
		return &MapDomainResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create port forward: %v", err),
		}, nil
	}

	settings.WebServer.ExposedDomain = pf.PublicURL
	_ = SaveSettings(settings)

	return &MapDomainResponse{
		Success:   pf.Status == portforward.StatusActive || pf.Status == portforward.StatusConnecting,
		Message:   fmt.Sprintf("Domain mapping initiated via %s", provider),
		PublicURL: pf.PublicURL,
	}, nil
}

// UnmapDomain removes the Cloudflare mapping for the web server.
func UnmapDomain() (*MapDomainResponse, error) {
	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}

	err = portforward.GetDefaultManager().Remove(settings.WebServer.Port)
	if err != nil {
		return &MapDomainResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to remove port forward: %v", err),
		}, nil
	}

	settings.WebServer.ExposedDomain = ""
	_ = SaveSettings(settings)

	return &MapDomainResponse{
		Success: true,
		Message: "Domain mapping removed",
	}, nil
}

// IsDomainMapped checks if the web server domain is currently mapped via portforward.
func IsDomainMapped() bool {
	settings, err := LoadSettings()
	if err != nil {
		return false
	}

	ports := portforward.GetDefaultManager().List()
	for _, pf := range ports {
		if pf.LocalPort == settings.WebServer.Port {
			return pf.Status == portforward.StatusActive || pf.Status == portforward.StatusConnecting
		}
	}
	return false
}
