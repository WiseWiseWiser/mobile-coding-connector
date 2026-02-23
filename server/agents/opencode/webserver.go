package opencode

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/portforward"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"

	exposed "github.com/xhd2015/lifelog-private/ai-critic/server/agents/opencode/exposed_opencode"
	"github.com/xhd2015/lifelog-private/ai-critic/server/proc_manager"
)

// WebServerProcessID is the ID used for managing the web server subprocess
const WebServerProcessID = "opencode-web-server"

// WebServerStatus represents the status of the OpenCode web server
type WebServerStatus struct {
	Running          bool   `json:"running"`
	Port             int    `json:"port"`
	Domain           string `json:"domain"`
	PortMapped       bool   `json:"port_mapped"`
	ConfigPath       string `json:"config_path"`
	AuthProxyRunning bool   `json:"auth_proxy_running"`
	AuthProxyFound   bool   `json:"auth_proxy_found"`
	AuthProxyPath    string `json:"auth_proxy_path"`
	OpencodePort     int    `json:"opencode_port"`
}

// GetWebServerStatus checks if the OpenCode web server is running and if its port is mapped
func GetWebServerStatus() (*WebServerStatus, error) {
	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}

	proxyPort := settings.WebServer.Port

	status := &WebServerStatus{
		Port:         proxyPort,
		Domain:       settings.DefaultDomain,
		OpencodePort: 0,
	}

	// Check if auth proxy binary exists and get its full path
	if path, err := tool_resolve.LookPath("basic-auth-proxy"); err == nil {
		status.AuthProxyFound = true
		status.AuthProxyPath = path
	} else {
		status.AuthProxyFound = false
		status.AuthProxyPath = ""
	}

	// Check if auth proxy is running on the proxy port
	if settings.WebServer.AuthProxyEnabled {
		status.AuthProxyRunning = IsAuthProxyRunning(proxyPort)
		// Get the actual opencode port from settings if running
		if status.AuthProxyRunning {
			status.OpencodePort = GetOpencodeInternalPort(proxyPort)
		}
		status.Running = status.AuthProxyRunning
	} else {
		// Check if web server is running by trying to connect to its port
		status.Running = IsWebServerRunning(settings.WebServer.Port)
		status.OpencodePort = settings.WebServer.Port
	}

	// Check if port is mapped to the domain
	if status.Running && settings.DefaultDomain != "" {
		status.PortMapped = isPortMappedToDomain(proxyPort, settings.DefaultDomain)
	}

	// Get config path
	home, err := os.UserHomeDir()
	if err == nil {
		status.ConfigPath = filepath.Join(home, ".local", "share", "opencode", "config.json")
	}

	return status, nil
}

// IsAuthProxyRunning checks if the auth proxy is running on the given port
func IsAuthProxyRunning(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// GetOpencodeInternalPort returns the internal opencode port from the proxy config file
// The proxy stores the backend port in its config
func GetOpencodeInternalPort(proxyPort int) int {
	configPath := filepath.Join(config.DataDir, "basic-auth-proxy.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0
	}
	var proxyCfg struct {
		BackendPort int `json:"backend_port"`
	}
	if err := json.Unmarshal(data, &proxyCfg); err != nil {
		return 0
	}
	return proxyCfg.BackendPort
}

// IsWebServerRunning checks if the OpenCode web server is running on the given port
func IsWebServerRunning(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
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
// It uses the exposed_opencode package for strict port and password handling.
func startWebServer(settings *Settings, customPath string) (*WebServerControlResponse, error) {
	port := settings.WebServer.Port
	if port == 0 {
		port = 4096
	}

	// If auth proxy is enabled, start opencode on a random available port and wrap with proxy
	if settings.WebServer.AuthProxyEnabled {
		return startWebServerWithProxy(settings, customPath, port)
	}

	server, err := exposed.StartWithSettings(port, settings.WebServer.Password, customPath)
	if err != nil {
		return &WebServerControlResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start web server: %v", err),
			Running: false,
		}, nil
	}

	return &WebServerControlResponse{
		Success: true,
		Message: fmt.Sprintf("Web server started successfully on port %d", server.Port),
		Running: true,
	}, nil
}

// startWebServerWithProxy starts opencode on a random port and wraps it with auth proxy
func startWebServerWithProxy(settings *Settings, customPath string, proxyPort int) (*WebServerControlResponse, error) {
	// Find an available port for opencode
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return &WebServerControlResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to find available port: %v", err),
			Running: false,
		}, nil
	}
	opencodePort := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	// Start opencode server on the available port
	server, err := exposed.StartWithSettings(opencodePort, settings.WebServer.Password, customPath)
	if err != nil {
		return &WebServerControlResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start opencode server: %v", err),
			Running: false,
		}, nil
	}

	// Save proxy config with backend port
	proxyConfigPath := filepath.Join(config.DataDir, "basic-auth-proxy.json")
	proxyCfg := struct {
		BackendPort int `json:"backend_port"`
	}{BackendPort: server.Port}
	proxyCfgData, _ := json.MarshalIndent(proxyCfg, "", "  ")
	os.WriteFile(proxyConfigPath, proxyCfgData, 0644)

	// Start the auth proxy
	err = startAuthProxy(proxyPort, server.Port)
	if err != nil {
		// Clean up opencode server
		exposed.Stop()
		return &WebServerControlResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start auth proxy: %v", err),
			Running: false,
		}, nil
	}

	return &WebServerControlResponse{
		Success: true,
		Message: fmt.Sprintf("Web server started successfully on port %d (with auth proxy)", proxyPort),
		Running: true,
	}, nil
}

const authProxyProcName = "basic-auth-proxy"

// startAuthProxy starts the basic-auth-proxy process
func startAuthProxy(proxyPort, backendPort int) error {
	// Check if binary exists
	if !tool_resolve.IsAvailable("basic-auth-proxy") {
		return fmt.Errorf("basic-auth-proxy binary not found in PATH")
	}

	cmd, err := tool_exec.New("basic-auth-proxy", []string{
		"--port", fmt.Sprintf("%d", proxyPort),
		"--backend-port", fmt.Sprintf("%d", backendPort),
	}, nil)
	if err != nil {
		return fmt.Errorf("failed to create proxy command: %w", err)
	}

	err = cmd.Cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start proxy: %w", err)
	}

	// Save to registry
	proc_manager.WithLock(authProxyProcName, func() error {
		err := proc_manager.SaveRegistry(authProxyProcName, &proc_manager.ProcessRegistry{
			Name:      authProxyProcName,
			PID:       cmd.Cmd.Process.Pid,
			Port:      proxyPort,
			StartTime: time.Now().Unix(),
		})
		if err != nil {
			fmt.Printf("[opencode] Warning: failed to save auth proxy registry: %v\n", err)
		}
		return nil
	})

	fmt.Printf("[opencode] Auth proxy started on port %d (backend: %d), PID: %d\n", proxyPort, backendPort, cmd.Cmd.Process.Pid)
	return nil
}

// stopAuthProxy stops the basic-auth-proxy process
func stopAuthProxy() error {
	var proxyPort int

	// Get proxy port from settings
	settings, err := LoadSettings()
	if err == nil {
		proxyPort = settings.WebServer.Port
	}

	// Try to stop using registry
	err = proc_manager.WithLock(authProxyProcName, func() error {
		reg, err := proc_manager.LoadRegistry(authProxyProcName)
		if err != nil {
			fmt.Printf("[opencode] Warning: failed to load auth proxy registry: %v\n", err)
		}

		if reg != nil && reg.PID > 0 && proc_manager.IsProcessAlive(reg.PID) {
			fmt.Printf("[opencode] Stopping auth proxy: PID=%d, Port=%d\n", reg.PID, reg.Port)
			proc_manager.StopProcess(reg.PID)
		}

		proc_manager.ClearRegistry(authProxyProcName)
		return nil
	})

	if err != nil {
		fmt.Printf("[opencode] Warning: failed to stop auth proxy via registry: %v\n", err)
	}

	// Fallback: try to kill by port if we have it
	if proxyPort > 0 {
		KillProcessByPort(proxyPort)
	}

	return nil
}

// stopWebServer attempts to stop the OpenCode web server.
// The customPath parameter allows using a user-configured binary path.
func stopWebServer(settings *Settings, customPath string) (*WebServerControlResponse, error) {
	port := settings.WebServer.Port

	// If auth proxy is enabled, stop it first
	if settings.WebServer.AuthProxyEnabled {
		stopAuthProxy()
		// Then stop the opencode server
		exposed.Stop()
		// Also try the standard opencode stop command as fallback
		cmd, err := tool_exec.New("opencode", []string{"web", "stop"}, &tool_exec.Options{
			CustomPath: customPath,
		})
		if err == nil {
			cmd.Cmd.Run()
		}

		// Wait a moment for the server to stop
		time.Sleep(1 * time.Second)

		// Check if opencode is still running on the internal port
		proxyConfigPath := filepath.Join(config.DataDir, "basic-auth-proxy.json")
		data, err := os.ReadFile(proxyConfigPath)
		opencodeRunning := false
		if err == nil {
			var cfg struct {
				BackendPort int `json:"backend_port"`
			}
			if err := json.Unmarshal(data, &cfg); err == nil && cfg.BackendPort > 0 {
				opencodeRunning = IsWebServerRunning(cfg.BackendPort)
				if opencodeRunning {
					fmt.Printf("Opencode server still running on port %d, attempting to kill process directly...\n", cfg.BackendPort)
					if err := KillProcessByPort(cfg.BackendPort); err != nil {
						fmt.Printf("Failed to kill opencode process: %v\n", err)
					}
				}
			}
		}

		// Remove proxy config
		os.Remove(proxyConfigPath)

		settings.WebServer.Enabled = false
		SaveSettings(settings)

		return &WebServerControlResponse{
			Success: true,
			Message: "Web server stopped successfully",
			Running: false,
		}, nil
	}

	// Stop using exposed_opencode package
	exposed.Stop()

	// Also try the standard opencode stop command as fallback
	cmd, err := tool_exec.New("opencode", []string{"web", "stop"}, &tool_exec.Options{
		CustomPath: customPath,
	})
	if err == nil {
		cmd.Cmd.Run()
	}

	// Wait a moment for the server to stop
	time.Sleep(1 * time.Second)

	// Check if it's now stopped
	running := IsWebServerRunning(port)

	// If still running, use pure Go implementation to kill the process by port
	if running {
		fmt.Printf("Web server still running on port %d, attempting to kill process directly...\n", port)
		if err := KillProcessByPort(port); err != nil {
			fmt.Printf("Failed to kill process by port: %v\n", err)
		} else {
			time.Sleep(500 * time.Millisecond)
			running = IsWebServerRunning(port)
		}
	}

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

// AutoStartWebServer adds the tunnel mapping for opencode web server if configured
// and attempts to start the server. The tunnel mapping is created regardless of
// whether the server is running, so the server can be started later and be accessible.
func AutoStartWebServer() {
	fmt.Printf("[opencode] AutoStartWebServer: BEGIN\n")

	settings, err := LoadSettings()
	if err != nil {
		fmt.Printf("[opencode] AutoStartWebServer: failed to load settings: %v\n", err)
		return
	}

	fmt.Printf("[opencode] AutoStartWebServer: loaded settings - DefaultDomain=%q, WebServer.Enabled=%v, WebServer.Port=%d\n",
		settings.DefaultDomain, settings.WebServer.Enabled, settings.WebServer.Port)

	if settings.DefaultDomain == "" {
		fmt.Printf("[opencode] AutoStartWebServer: no default domain configured, skipping\n")
		return
	}

	port := settings.WebServer.Port
	if port == 0 {
		port = 4096
	}

	// Check if already running (for logging only, don't skip start)
	isRunning := IsWebServerRunning(port)
	fmt.Printf("[opencode] AutoStartWebServer: web server running on port %d? %v\n", port, isRunning)

	// Ensure extension tunnel group has a tunnel configured (reuses existing if domains already created one)
	tg := cloudflare.GetTunnelGroupManager().GetExtensionGroup()
	logFn := func(msg string) {
		fmt.Printf("[opencode] AutoStartWebServer: %s\n", msg)
	}

	tunnelRef, _, _, err := cloudflare.EnsureGroupTunnelConfigured(cloudflare.GroupExtension, "", logFn)
	if err != nil {
		fmt.Printf("[opencode] AutoStartWebServer: failed to ensure extension tunnel configured: %v\n", err)
		return
	}

	// Create DNS route for the domain (if not already exists)
	fmt.Printf("[opencode] AutoStartWebServer: ensuring DNS route for %s...\n", settings.DefaultDomain)
	if err := cloudflare.CreateDNSRoute(tunnelRef, settings.DefaultDomain); err != nil {
		fmt.Printf("[opencode] AutoStartWebServer: warning: DNS route error: %v\n", err)
	} else {
		fmt.Printf("[opencode] AutoStartWebServer: DNS route created or already exists\n")
	}

	utmMappings := tg.ListMappings()
	fmt.Printf("[opencode] AutoStartWebServer: extension tunnel group has %d mappings\n", len(utmMappings))
	for i, m := range utmMappings {
		fmt.Printf("[opencode] AutoStartWebServer:   mapping[%d] ID=%s, Hostname=%s, Service=%s\n", i, m.ID, m.Hostname, m.Service)
	}

	hasMapping := false
	for _, m := range utmMappings {
		// Check if this mapping points to the opencode web server port
		// Expected service format: http://localhost:PORT or http://127.0.0.1:PORT
		if strings.EqualFold(m.Hostname, settings.DefaultDomain) {
			servicePort := extractPortFromService(m.Service)
			fmt.Printf("[opencode] AutoStartWebServer: found matching hostname %s, service=%s, extractedPort=%d, configuredPort=%d\n",
				m.Hostname, m.Service, servicePort, port)
			if servicePort == port {
				hasMapping = true
				fmt.Printf("[opencode] AutoStartWebServer: mapping already exists with correct port\n")
				break
			}
		}
	}

	// Add mapping if needed
	if !hasMapping {
		serviceURL := fmt.Sprintf("http://localhost:%d", port)
		mappingID := fmt.Sprintf("port-%d", port)
		mapping := &cloudflare.IngressMapping{
			ID:       mappingID,
			Hostname: settings.DefaultDomain,
			Service:  serviceURL,
			Source:   "opencode-autostart",
		}
		fmt.Printf("[opencode] AutoStartWebServer: adding mapping ID=%s, Hostname=%s, Service=%s\n",
			mapping.ID, mapping.Hostname, mapping.Service)
		if err := tg.AddMapping(mapping); err != nil {
			fmt.Printf("[opencode] AutoStartWebServer: failed to add mapping to extension tunnel: %v\n", err)
			return
		}
		fmt.Printf("[opencode] AutoStartWebServer: mapping added successfully\n")
	}

	// Try to start the web server (non-blocking, will succeed if already running)
	go func() {
		fmt.Printf("[opencode] AutoStartWebServer: attempting to start web server for domain %s...\n", settings.DefaultDomain)
		resp, err := ControlWebServer("start", "")
		if err != nil {
			fmt.Printf("[opencode] AutoStartWebServer: ControlWebServer returned error: %v\n", err)
		} else if resp != nil {
			fmt.Printf("[opencode] AutoStartWebServer: ControlWebServer result - Success=%v, Message=%q, Running=%v\n",
				resp.Success, resp.Message, resp.Running)
		} else {
			fmt.Printf("[opencode] AutoStartWebServer: ControlWebServer returned nil response\n")
		}
	}()
}

// extractPortFromService extracts the port number from a service URL
// e.g., "http://localhost:4096" -> 4096, "http://127.0.0.1:8080" -> 8080
func extractPortFromService(service string) int {
	if idx := strings.LastIndex(service, ":"); idx != -1 {
		portStr := service[idx+1:]
		port, err := strconv.Atoi(portStr)
		if err == nil {
			return port
		}
	}
	return 0
}

// IsWebServerMapping checks if a mapping ID belongs to the opencode web server
func IsWebServerMapping(mappingID string) bool {
	settings, err := LoadSettings()
	if err != nil {
		return false
	}

	port := settings.WebServer.Port
	if port == 0 {
		port = 4096
	}

	// Mapping ID format is "port-<port>" for port forward mappings
	expectedID := fmt.Sprintf("port-%d", port)
	return mappingID == expectedID
}
