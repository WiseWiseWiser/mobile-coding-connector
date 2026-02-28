package exposed_opencode

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/proxy/basic_auth_proxy"
)

// WebServerProcessID is the ID used for managing the web server subprocess.
const WebServerProcessID = "opencode-web-server"

// WebServerControlRequest represents a request to start/stop the web server.
type WebServerControlRequest struct {
	Action string `json:"action"` // "start" or "stop"
}

// StartWebServer starts the OpenCode web server.
func StartWebServer() (*WebServerControlResponse, error) {
	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}
	return startWebServer(settings)
}

// StopWebServer stops the OpenCode web server.
func StopWebServer() (*WebServerControlResponse, error) {
	settings, err := LoadSettings()
	if err != nil {
		return nil, err
	}
	return stopWebServer(settings)
}

func startWebServer(settings *Settings) (*WebServerControlResponse, error) {
	port := settings.WebServer.Port
	if port == 0 {
		port = 4096
	}

	if settings.WebServer.AuthProxyEnabled {
		return startWebServerWithProxy(settings, port)
	}

	server, err := StartWithSettings(port, settings.WebServer.Password, settings.BinaryPath)
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

func startWebServerWithProxy(settings *Settings, proxyPort int) (*WebServerControlResponse, error) {
	server, err := StartWithSettings(0, settings.WebServer.Password, settings.BinaryPath)
	if err != nil {
		return &WebServerControlResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start opencode server: %v", err),
			Running: false,
		}, nil
	}

	if err := basic_auth_proxy.Start(proxyPort, server.Port); err != nil {
		Stop()
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

func stopWebServer(settings *Settings) (*WebServerControlResponse, error) {
	port := settings.WebServer.Port

	if settings.WebServer.AuthProxyEnabled {
		backendPort := basic_auth_proxy.GetBackendPort()
		if err := basic_auth_proxy.Stop(); err != nil {
			fmt.Printf("[opencode] Warning: failed to stop auth proxy via registry: %v\n", err)
		}

		Stop()
		if backendPort > 0 {
			if err := forceStopProcessOnPort(backendPort); err != nil {
				fmt.Printf("[opencode] Warning: failed to force-stop backend on port %d: %v\n", backendPort, err)
			}
		}
		if port > 0 {
			if err := forceStopProcessOnPort(port); err != nil {
				fmt.Printf("[opencode] Warning: failed to force-stop auth proxy on port %d: %v\n", port, err)
			}
		}

		backendStopped := true
		if backendPort > 0 {
			backendStopped = waitForWebServerStop(backendPort, 5*time.Second)
		}
		proxyStopped := true
		if port > 0 {
			proxyStopped = waitForWebServerStop(port, 5*time.Second)
		}
		if !backendStopped && backendPort > 0 {
			fmt.Printf("[opencode] Warning: backend server still appears to be running on port %d\n", backendPort)
		}

		if err := basic_auth_proxy.RemoveConfig(); err != nil {
			fmt.Printf("[opencode] Warning: failed to remove auth proxy config: %v\n", err)
		}

		running := !(backendStopped && proxyStopped)

		return &WebServerControlResponse{
			Success: !running,
			Message: func() string {
				if !running {
					return "Web server stopped successfully"
				}
				return "Web server may still be running"
			}(),
			Running: running,
		}, nil
	}

	Stop()
	if err := forceStopProcessOnPort(port); err != nil {
		fmt.Printf("[opencode] Warning: failed to force-stop server on port %d: %v\n", port, err)
	}

	running := !waitForWebServerStop(port, 5*time.Second)
	if running {
		fmt.Printf("[opencode] Warning: web server still appears to be running on port %d after stop command\n", port)
	}

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

func waitForWebServerStop(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !IsWebServerRunning(port) {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return !IsWebServerRunning(port)
}

func forceStopProcessOnPort(port int) error {
	if port <= 0 {
		return nil
	}

	out, err := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port)).Output()
	if err != nil {
		// lsof exits non-zero when no process is bound to the port.
		if _, ok := err.(*exec.ExitError); ok {
			return nil
		}
		return err
	}

	pids := strings.Fields(strings.TrimSpace(string(out)))
	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	return nil
}

// AutoStartWebServer adds the tunnel mapping for opencode web server if configured.
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

	isRunning := IsWebServerRunning(port)
	fmt.Printf("[opencode] AutoStartWebServer: web server running on port %d? %v\n", port, isRunning)

	tg := cloudflare.GetTunnelGroupManager().GetExtensionGroup()
	logFn := func(msg string) {
		fmt.Printf("[opencode] AutoStartWebServer: %s\n", msg)
	}

	tunnelRef, _, _, err := cloudflare.EnsureGroupTunnelConfigured(cloudflare.GroupExtension, "", logFn)
	if err != nil {
		fmt.Printf("[opencode] AutoStartWebServer: failed to ensure extension tunnel configured: %v\n", err)
		return
	}

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

	go func() {
		fmt.Printf("[opencode] AutoStartWebServer: attempting to start web server for domain %s...\n", settings.DefaultDomain)
		resp, err := StartWebServer()
		if err != nil {
			fmt.Printf("[opencode] AutoStartWebServer: StartWebServer returned error: %v\n", err)
		} else if resp != nil {
			fmt.Printf("[opencode] AutoStartWebServer: StartWebServer result - Success=%v, Message=%q, Running=%v\n",
				resp.Success, resp.Message, resp.Running)
		} else {
			fmt.Printf("[opencode] AutoStartWebServer: StartWebServer returned nil response\n")
		}
	}()
}

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

// IsWebServerMapping checks if a mapping ID belongs to the opencode web server.
func IsWebServerMapping(mappingID string) bool {
	settings, err := LoadSettings()
	if err != nil {
		return false
	}

	port := settings.WebServer.Port
	if port == 0 {
		port = 4096
	}

	expectedID := fmt.Sprintf("port-%d", port)
	return mappingID == expectedID
}
