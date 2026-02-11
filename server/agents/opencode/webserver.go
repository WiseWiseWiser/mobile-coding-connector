package opencode

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/portforward"
	"github.com/xhd2015/lifelog-private/ai-critic/server/subprocess"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
)

// WebServerProcessID is the ID used for managing the web server subprocess
const WebServerProcessID = "opencode-web-server"

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
	status.Running = IsWebServerRunning(settings.WebServer.Port)

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

// IsWebServerRunning checks if the OpenCode web server is running on the given port
func IsWebServerRunning(port int) bool {
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

	if !IsWebServerRunning(port) {
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

// getBaseDomain extracts the base domain from a full domain
// e.g., "x.y.com" -> "y.com", "sub.example.co.uk" -> "example.co.uk"
func getBaseDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return ""
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return domain
	}

	// For domains like "example.co.uk", we need to handle them specially
	// This is a simplified approach - returns the last 2 parts
	// More complex logic would check against a list of public suffixes
	if len(parts) >= 3 {
		// Check if second-to-last part is a common second-level domain like "co", "com", "org", "gov", etc.
		secondLevel := parts[len(parts)-2]
		if secondLevel == "co" || secondLevel == "com" || secondLevel == "org" || secondLevel == "gov" || secondLevel == "edu" || secondLevel == "net" || secondLevel == "ac" {
			// Include 3 parts for domains like example.co.uk
			return strings.Join(parts[len(parts)-3:], ".")
		}
	}

	// Default: return last 2 parts
	return strings.Join(parts[len(parts)-2:], ".")
}

// DomainMatchesOwned checks if the configured domain's base domain matches any owned domain
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

// WebServerControlRequest represents a request to start/stop the web server
type WebServerControlRequest struct {
	Action string `json:"action"` // "start" or "stop"
}

// WebServerControlResponse represents the response from a control operation
type WebServerControlResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Running bool   `json:"running"`
}

// ControlWebServer starts or stops the OpenCode web server.
// The customPath parameter, if provided, will be used as the opencode binary path.
// This allows using user-configured paths from agent settings.
func ControlWebServer(action string, customPath string) (*WebServerControlResponse, error) {
	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}

	switch action {
	case "start":
		return startWebServer(settings, customPath)
	case "stop":
		return stopWebServer(settings, customPath)
	default:
		return nil, fmt.Errorf("invalid action: %s (must be 'start' or 'stop')", action)
	}
}

// startWebServer attempts to start the OpenCode web server.
// The customPath parameter allows using a user-configured binary path.
func startWebServer(settings *Settings, customPath string) (*WebServerControlResponse, error) {
	// Check if already running via subprocess manager
	manager := subprocess.GetManager()
	if manager.IsRunning(WebServerProcessID) {
		return &WebServerControlResponse{
			Success: true,
			Message: fmt.Sprintf("Web server is already running on port %d", settings.WebServer.Port),
			Running: true,
		}, nil
	}

	// Also check via HTTP health check
	if IsWebServerRunning(settings.WebServer.Port) {
		return &WebServerControlResponse{
			Success: true,
			Message: fmt.Sprintf("Web server is already running on port %d", settings.WebServer.Port),
			Running: true,
		}, nil
	}

	// Create command using tool_exec for proper PATH resolution
	// Pass password via environment variable if configured
	cmdOpts := &tool_exec.Options{
		CustomPath: customPath,
	}
	if settings.WebServer.Password != "" {
		cmdOpts.Env = map[string]string{
			"OPENCODE_SERVER_PASSWORD": settings.WebServer.Password,
		}
	}
	cmd, err := tool_exec.New("opencode", []string{"web", "--port", fmt.Sprintf("%d", settings.WebServer.Port)}, cmdOpts)
	if err != nil {
		return &WebServerControlResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create web server command: %v", err),
			Running: false,
		}, nil
	}

	// Health checker function
	healthChecker := func() bool {
		return IsWebServerRunning(settings.WebServer.Port)
	}

	// Start the process via subprocess manager (non-blocking)
	process, err := manager.StartProcess(WebServerProcessID, "OpenCode Web Server", cmd.Cmd, healthChecker)
	if err != nil {
		return &WebServerControlResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start web server: %v", err),
			Running: false,
		}, nil
	}

	// Wait for the server to be ready (with timeout)
	running := process.WaitForRunning(10 * time.Second)

	// Update settings to mark web server as enabled
	settings.WebServer.Enabled = running
	SaveSettings(settings)

	return &WebServerControlResponse{
		Success: running,
		Message: func() string {
			if running {
				return fmt.Sprintf("Web server started successfully on port %d", settings.WebServer.Port)
			}
			return "Web server process started but health check failed"
		}(),
		Running: running,
	}, nil
}

// stopWebServer attempts to stop the OpenCode web server.
// The customPath parameter allows using a user-configured binary path.
func stopWebServer(settings *Settings, customPath string) (*WebServerControlResponse, error) {
	manager := subprocess.GetManager()

	// First try to stop via subprocess manager if it's managed
	if manager.IsRunning(WebServerProcessID) {
		if err := manager.StopProcess(WebServerProcessID); err != nil {
			// If subprocess manager fails, try the opencode stop command
			fmt.Printf("Subprocess manager stop failed: %v, trying opencode web stop\n", err)
		}
	}

	// Also try the standard opencode stop command
	cmd, err := tool_exec.New("opencode", []string{"web", "stop"}, &tool_exec.Options{
		CustomPath: customPath,
	})
	if err == nil {
		// Run stop command (ignore errors, it might not be running)
		cmd.Cmd.Run()
	}

	// Wait a moment for the server to stop
	time.Sleep(1 * time.Second)

	// Check if it's now stopped
	running := IsWebServerRunning(settings.WebServer.Port)

	// Update settings to mark web server as disabled
	settings.WebServer.Enabled = running
	SaveSettings(settings)

	return &WebServerControlResponse{
		Success: !running,
		Message: func() string {
			if !running {
				return "Web server stopped successfully"
			}
			return "Web server stop command executed but server may still be running"
		}(),
		Running: running,
	}, nil
}

// MapDomainRequest represents a request to map the domain via Cloudflare
type MapDomainRequest struct {
	Provider string `json:"provider,omitempty"` // Optional: "cloudflare_owned" or "cloudflare_tunnel"
}

// MapDomainResponse represents the response from a domain mapping operation
type MapDomainResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	PublicURL string `json:"public_url,omitempty"`
}

// MapDomainViaCloudflare maps the web server port to the configured domain using Cloudflare
// This reuses the same portforward manager as the Ports tab
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

	// Check if domain matches an owned domain
	matches, _ := DomainMatchesOwned(settings.DefaultDomain)
	if !matches {
		return &MapDomainResponse{
			Success: false,
			Message: fmt.Sprintf("Domain %s does not match any owned domain (base: %s)", settings.DefaultDomain, getBaseDomain(settings.DefaultDomain)),
		}, nil
	}

	// Default to cloudflare_owned if not specified
	if provider == "" {
		provider = portforward.ProviderCloudflareOwned
	}

	// Create a label for this port forward
	label := settings.DefaultDomain

	// Use the portforward manager to create the tunnel
	// We need to access the default manager from the portforward package
	pf, err := portforward.GetDefaultManager().Add(settings.WebServer.Port, label, provider)
	if err != nil {
		return &MapDomainResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create port forward: %v", err),
		}, nil
	}

	// Save the exposed domain
	settings.WebServer.ExposedDomain = pf.PublicURL
	SaveSettings(settings)

	return &MapDomainResponse{
		Success:   pf.Status == portforward.StatusActive || pf.Status == portforward.StatusConnecting,
		Message:   fmt.Sprintf("Domain mapping initiated via %s", provider),
		PublicURL: pf.PublicURL,
	}, nil
}

// UnmapDomain removes the Cloudflare mapping for the web server
func UnmapDomain() (*MapDomainResponse, error) {
	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}

	// Use the portforward manager to remove the tunnel
	err = portforward.GetDefaultManager().Remove(settings.WebServer.Port)
	if err != nil {
		return &MapDomainResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to remove port forward: %v", err),
		}, nil
	}

	// Clear the exposed domain
	settings.WebServer.ExposedDomain = ""
	SaveSettings(settings)

	return &MapDomainResponse{
		Success: true,
		Message: "Domain mapping removed",
	}, nil
}

// IsDomainMapped checks if the web server domain is currently mapped via portforward
func IsDomainMapped() bool {
	settings, err := LoadSettings()
	if err != nil {
		return false
	}

	// Check if this port is being forwarded
	ports := portforward.GetDefaultManager().List()
	for _, pf := range ports {
		if pf.LocalPort == settings.WebServer.Port {
			return pf.Status == portforward.StatusActive || pf.Status == portforward.StatusConnecting
		}
	}

	return false
}
