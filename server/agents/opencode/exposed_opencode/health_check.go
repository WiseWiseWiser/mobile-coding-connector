package exposed_opencode

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

var (
	healthCheckStopChan chan struct{}
	healthCheckRunning  int32 // atomic: 0 = not running, 1 = running
)

// StartHealthCheck starts the health check loop that runs every 10 seconds.
func StartHealthCheck() {
	if atomic.LoadInt32(&healthCheckRunning) == 1 {
		return
	}
	atomic.StoreInt32(&healthCheckRunning, 1)
	healthCheckStopChan = make(chan struct{})

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				settings, err := LoadSettings()
				if err != nil {
					fmt.Printf("[opencode] Health check: failed to load settings: %v\n", err)
					continue
				}

				if !settings.WebServer.Enabled {
					continue
				}

				configuredPort := settings.WebServer.Port
				if configuredPort == 0 {
					configuredPort = 4096
				}

				if !isWebServerReachable(configuredPort) {
					fmt.Printf("[opencode] Health check: port %d not reachable, restarting server...\n", configuredPort)
					if _, err := StopWebServer(); err != nil {
						fmt.Printf("[opencode] Health check: stop failed: %v\n", err)
					}
					resp, err := StartWebServer()
					if err != nil {
						fmt.Printf("[opencode] Health check: start failed: %v\n", err)
						continue
					}
					if resp != nil && !resp.Success {
						fmt.Printf("[opencode] Health check: restart reported failure: %s\n", resp.Message)
					}
				}
			case <-healthCheckStopChan:
				fmt.Println("[opencode] Health check: stopping...")
				return
			}
		}
	}()

	fmt.Println("[opencode] Health check: started (every 10s)")
}

// StopHealthCheck stops the health check loop.
func StopHealthCheck() {
	if atomic.LoadInt32(&healthCheckRunning) == 0 {
		return
	}
	atomic.StoreInt32(&healthCheckRunning, 0)
	if healthCheckStopChan != nil {
		close(healthCheckStopChan)
		healthCheckStopChan = nil
	}
}

func isWebServerReachable(port int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/session", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}
