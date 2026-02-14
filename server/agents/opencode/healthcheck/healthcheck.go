// Package healthcheck provides health checking and restart functionality for the OpenCode web server.
package healthcheck

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

// ServerConfig holds the configuration for health checking and restarting the server.
type ServerConfig struct {
	Port       int
	CustomPath string
	Password   string
	WorkDir    string
}

// CheckPortReachable checks if the opencode server is reachable on the given port.
func CheckPortReachable(port int) bool {
	url := fmt.Sprintf("http://127.0.0.1:%d/session", port)
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	// Accept any response (200 or 401) as "reachable"
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}

// WaitForServer waits for the server to be ready on the given port.
func WaitForServer(port int, timeout time.Duration) error {
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

// IsPortReachable is an alias for CheckPortReachable for backward compatibility.
func IsPortReachable(port int) bool {
	return CheckPortReachable(port)
}

// FindAvailablePort finds an available TCP port.
func FindAvailablePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port, nil
}
