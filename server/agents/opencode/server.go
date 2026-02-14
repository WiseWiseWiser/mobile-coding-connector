package opencode

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_exec"
)

var (
	serverInstance      *OpencodeServer
	serverMutex         sync.Mutex
	starting            int32 // atomic: 0 = not starting, 1 = starting
	healthCheckStopChan chan struct{}
	healthCheckRunning  int32 // atomic: 0 = not running, 1 = running
)

// OpencodeServer holds the state of a running opencode server
type OpencodeServer struct {
	Port     int
	Cmd      *exec.Cmd
	StopChan chan struct{}
}

// GetOrStartOpencodeServer returns the existing opencode server or starts a new one
// It will reuse an already started server, or start a new one if none exists
func GetOrStartOpencodeServer() (*OpencodeServer, error) {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if serverInstance != nil && serverInstance.Cmd != nil && serverInstance.Cmd.Process != nil {
		return serverInstance, nil
	}

	if serverInstance != nil && serverInstance.StopChan != nil {
		select {
		case <-serverInstance.StopChan:
		default:
			close(serverInstance.StopChan)
		}
		serverInstance = nil
	}

	if atomic.LoadInt32(&starting) == 1 {
		for atomic.LoadInt32(&starting) == 1 {
			time.Sleep(100 * time.Millisecond)
		}
		return serverInstance, nil
	}

	atomic.StoreInt32(&starting, 1)
	defer atomic.StoreInt32(&starting, 0)

	port, err := findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port: %w", err)
	}

	serverInstance = &OpencodeServer{
		Port:     port,
		StopChan: make(chan struct{}),
	}

	if err := startOpencodeWebServer(serverInstance); err != nil {
		return nil, fmt.Errorf("failed to start opencode server: %w", err)
	}

	if err := waitForServer(port, 10*time.Second); err != nil {
		return nil, fmt.Errorf("opencode server not ready: %w", err)
	}

	fmt.Printf("[opencode] Server started on port %d\n", port)
	return serverInstance, nil
}

func findAvailablePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port, nil
}

func startOpencodeWebServer(server *OpencodeServer) error {
	// Use tool_exec for proper PATH resolution
	cmdWrapper, err := tool_exec.New("opencode", []string{"web", "--port", fmt.Sprintf("%d", server.Port)}, &tool_exec.Options{
		Dir: "/root/mobile-coding-connector",
	})
	if err != nil {
		return fmt.Errorf("failed to create opencode command: %w", err)
	}

	cmd := cmdWrapper.Cmd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return err
	}

	server.Cmd = cmd

	// Handle process exit
	go func() {
		select {
		case <-server.StopChan:
			if server.Cmd != nil && server.Cmd.Process != nil {
				server.Cmd.Process.Kill()
			}
		case <-waitDone(server.Cmd):
			// Process exited
		}
	}()

	return nil
}

func waitForServer(port int, timeout time.Duration) error {
	url := fmt.Sprintf("http://127.0.0.1:%d/session", port)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			// Accept any response (200 or 401) as "ready"
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for server on port %d", port)
}

func waitDone(cmd *exec.Cmd) chan struct{} {
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()
	return done
}

// ShutdownOpencodeServer stops the opencode server
func ShutdownOpencodeServer() {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if serverInstance != nil && serverInstance.StopChan != nil {
		fmt.Printf("[opencode] Stopping server on port %d\n", serverInstance.Port)
		close(serverInstance.StopChan)
		serverInstance.Cmd = nil
	}
}

// GetRunningServerPort returns the port of the currently running server, or 0 if not running
func GetRunningServerPort() int {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	if serverInstance != nil && serverInstance.Cmd != nil && serverInstance.Cmd.Process != nil {
		return serverInstance.Port
	}
	return 0
}

// isPortReachable checks if the opencode server is reachable on the given port
func isPortReachable(port int) bool {
	url := fmt.Sprintf("http://127.0.0.1:%d/session", port)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	// Accept any response (200 or 401) as "reachable"
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}

// restartOpencodeServerWithoutTunnel restarts only the opencode server (not the tunnel mapping)
// It uses the configured port and binary path from settings, reusing the same logic as the "Start" button.
func restartOpencodeServerWithoutTunnel() error {
	// Load settings to get configured port and binary path
	settings, err := LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Use the configured port from settings
	port := settings.WebServer.Port
	if port == 0 {
		port = 4096 // default port
	}

	// Clean up old instance
	ShutdownOpencodeServer()

	// Use the same function as the "Start" button to start the server
	// This will automatically use the configured binary path from agent settings
	// because startWebServer uses tool_exec which respects the CustomPath option
	_, err = startWebServer(settings, "")
	if err != nil {
		return fmt.Errorf("failed to restart opencode server: %w", err)
	}

	fmt.Printf("[opencode] Health check: Server restarted on port %d\n", port)
	return nil
}

// getAgentBinaryPathForOpencode returns the user-configured binary path for opencode
func getAgentBinaryPathForOpencode() string {
	// Import the agents package to get the binary path
	// We can't import agents package here due to import cycle, so we use a different approach
	// The binary path will be passed through the settings or we need to read it from the config
	// For now, return empty string to use default "opencode" command
	// The actual custom path should be passed from the calling code
	return ""
}

// startOpencodeWebServerWithConfig starts the opencode web server with a custom binary path
func startOpencodeWebServerWithConfig(server *OpencodeServer, customPath string) error {
	// Load settings to get password if configured
	settings, err := LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Create command using tool_exec for proper PATH resolution
	cmdOpts := &tool_exec.Options{
		CustomPath: customPath,
	}

	// Pass password via environment variable if configured
	if settings.WebServer.Password != "" {
		cmdOpts.Env = map[string]string{
			"OPENCODE_SERVER_PASSWORD": settings.WebServer.Password,
		}
	}

	cmdWrapper, err := tool_exec.New("opencode", []string{"web", "--port", fmt.Sprintf("%d", server.Port)}, cmdOpts)
	if err != nil {
		return fmt.Errorf("failed to create opencode command: %w", err)
	}

	cmd := cmdWrapper.Cmd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return err
	}

	server.Cmd = cmd

	// Handle process exit
	go func() {
		select {
		case <-server.StopChan:
			if server.Cmd != nil && server.Cmd.Process != nil {
				server.Cmd.Process.Kill()
			}
		case <-waitDone(cmd):
			// Process exited
		}
	}()

	return nil
}

// StartHealthCheck starts the health check loop that runs every 10 seconds
func StartHealthCheck() {
	if atomic.LoadInt32(&healthCheckRunning) == 1 {
		return // Already running
	}
	atomic.StoreInt32(&healthCheckRunning, 1)
	healthCheckStopChan = make(chan struct{})

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Load settings to get configured port and check if web server is enabled
				settings, err := LoadSettings()
				if err != nil {
					fmt.Printf("[opencode] Health check: Failed to load settings: %v\n", err)
					continue
				}

				// Only check/restart if web server is enabled
				if !settings.WebServer.Enabled {
					continue
				}

				// Use the configured port from settings
				configuredPort := settings.WebServer.Port
				if configuredPort == 0 {
					configuredPort = 4096 // default port
				}

				// Check if server is running on the configured port
				if !isPortReachable(configuredPort) {
					// Server is not reachable, try to restart it
					fmt.Printf("[opencode] Health check: Port %d not reachable, restarting server...\n", configuredPort)

					// Use restartOpencodeServerWithoutTunnel which will use the configured settings
					if err := restartOpencodeServerWithoutTunnel(); err != nil {
						fmt.Printf("[opencode] Health check: Failed to restart server: %v\n", err)
					}
				}
			case <-healthCheckStopChan:
				fmt.Println("[opencode] Health check: Stopping...")
				return
			}
		}
	}()

	fmt.Println("[opencode] Health check: Started (checking every 10s)")
}

// StopHealthCheck stops the health check loop
func StopHealthCheck() {
	if atomic.LoadInt32(&healthCheckRunning) == 0 {
		return // Not running
	}
	atomic.StoreInt32(&healthCheckRunning, 0)
	if healthCheckStopChan != nil {
		close(healthCheckStopChan)
		healthCheckStopChan = nil
	}
}
