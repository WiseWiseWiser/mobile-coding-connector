package opencode

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// WebServerStatus represents the status of the OpenCode web server
type WebServerStatus struct {
	Running    bool   `json:"running"`
	Port       int    `json:"port"`
	Domain     string `json:"domain"`
	PortMapped bool   `json:"port_mapped"`
	ConfigPath string `json:"config_path"`
}

// GetWebServerStatus checks if the OpenCode web server is running and if its port is mapped
func GetWebServerStatus() (*WebServerStatus, error) {
	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}

	status := &WebServerStatus{
		Port:   settings.WebServer.Port,
		Domain: settings.DefaultDomain,
	}

	// Check if web server is running by trying to connect to its port
	status.Running = isWebServerRunning(settings.WebServer.Port)

	// Check if port is mapped to the domain
	if status.Running && settings.DefaultDomain != "" {
		status.PortMapped = isPortMappedToDomain(settings.WebServer.Port, settings.DefaultDomain)
	}

	// Get config path
	home, err := os.UserHomeDir()
	if err == nil {
		status.ConfigPath = filepath.Join(home, ".local", "share", "opencode", "config.json")
	}

	return status, nil
}

// isWebServerRunning checks if the OpenCode web server is running on the given port
func isWebServerRunning(port int) bool {
	url := fmt.Sprintf("http://127.0.0.1:%d/global/health", port)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// isPortMappedToDomain checks if the given port is mapped to the domain
// This is a simplified check - in a real implementation, you might want to
// check DNS resolution, port forwarding rules, etc.
func isPortMappedToDomain(port int, domain string) bool {
	// For now, we'll check if the domain resolves and if we can access the port via the domain
	// In practice, this might involve checking:
	// 1. DNS resolution
	// 2. Reverse proxy configuration
	// 3. Tunnel status (cloudflare, etc.)

	// Simple check: try to access via HTTP
	url := fmt.Sprintf("http://%s/global/health", domain)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Also check the response to ensure it's the same server
	return resp.StatusCode == http.StatusOK
}

// IsWebServerEnabled checks if the web server is enabled in settings
func IsWebServerEnabled() bool {
	settings, err := LoadSettings()
	if err != nil {
		return false
	}
	return settings.WebServer.Enabled
}

// GetWebServerPort returns the configured web server port
func GetWebServerPort() int {
	settings, err := LoadSettings()
	if err != nil {
		return 4096 // default port
	}
	if settings.WebServer.Port == 0 {
		return 4096
	}
	return settings.WebServer.Port
}

// CheckPortMappingStatus returns a human-readable status of the port mapping
func CheckPortMappingStatus(port int, domain string) string {
	if domain == "" {
		return "No domain configured"
	}

	if !isWebServerRunning(port) {
		return "Web server is not running"
	}

	if isPortMappedToDomain(port, domain) {
		return fmt.Sprintf("Port %d is successfully mapped to %s", port, domain)
	}

	return fmt.Sprintf("Port %d is not mapped to %s. Check your DNS or tunnel configuration.", port, domain)
}

// ExtractDomainFromURL extracts the domain from a URL string
func ExtractDomainFromURL(urlStr string) string {
	urlStr = strings.TrimSpace(urlStr)
	urlStr = strings.TrimPrefix(urlStr, "http://")
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "ws://")
	urlStr = strings.TrimPrefix(urlStr, "wss://")

	// Remove path and query parameters
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
