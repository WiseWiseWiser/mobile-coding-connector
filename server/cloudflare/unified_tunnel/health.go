package unified_tunnel

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// CloudflareErrorOriginDown is Cloudflare 530 / error 1033: the edge is
// reachable but no tunnel connector is registered. Restarting cloudflared
// on this code resets QUIC→HTTP/2 fallback and keeps the hostname dead.
const CloudflareErrorOriginDown = 530

type healthProbe struct {
	code int
	err  error
	at   time.Time
}

var (
	healthCheckHTTPClient = &http.Client{Timeout: 10 * time.Second}
	healthCheckBaseURL    string
	healthCheckBaseMu     sync.RWMutex
)

// IsHealthCheckHealthy reports whether a public GET should keep the tunnel
// process alive. 2xx–4xx mean visitors reached an origin; 530 means the
// edge is up but the connector is still registering — do not SIGTERM.
func IsHealthCheckHealthy(statusCode int) bool {
	return (statusCode >= 200 && statusCode < 500) || statusCode == CloudflareErrorOriginDown
}

// IsEdgeRegistered is true when visitors can reach an origin through the
// tunnel (not merely that Cloudflare returned a page). 530 is not registered.
func IsEdgeRegistered(statusCode int) bool {
	return statusCode >= 200 && statusCode < 500
}

// SetHealthCheckBaseURLForTest rewrites health-check URLs to an httptest
// server. Returns a restore func.
func SetHealthCheckBaseURLForTest(base string) func() {
	healthCheckBaseMu.Lock()
	prev := healthCheckBaseURL
	healthCheckBaseURL = base
	healthCheckBaseMu.Unlock()
	return func() {
		healthCheckBaseMu.Lock()
		healthCheckBaseURL = prev
		healthCheckBaseMu.Unlock()
	}
}

func healthCheckURLs(hostname string) []string {
	healthCheckBaseMu.RLock()
	base := healthCheckBaseURL
	healthCheckBaseMu.RUnlock()
	if base != "" {
		return []string{base + "/", base + "/ping"}
	}
	return []string{
		fmt.Sprintf("https://%s/", hostname),
		fmt.Sprintf("https://%s/ping", hostname),
	}
}

// ProbeHostname GETs / and /ping on hostname and returns the first useful
// status code. A network error with no HTTP response is (0, err).
func ProbeHostname(hostname string) (int, error) {
	var lastErr error
	var lastCode int
	for _, url := range healthCheckURLs(hostname) {
		resp, err := healthCheckHTTPClient.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		code := resp.StatusCode
		resp.Body.Close()
		lastCode = code
		lastErr = nil
		if IsHealthCheckHealthy(code) || IsEdgeRegistered(code) {
			return code, nil
		}
	}
	return lastCode, lastErr
}

func (utm *UnifiedTunnelManager) recordHealthProbe(hostname string, code int, err error) {
	utm.mu.Lock()
	defer utm.mu.Unlock()
	if utm.lastHealthProbe == nil {
		utm.lastHealthProbe = make(map[string]healthProbe)
	}
	utm.lastHealthProbe[hostname] = healthProbe{code: code, err: err, at: time.Now()}
}

// LastHealthProbe returns the last public GET status recorded for hostname.
func (utm *UnifiedTunnelManager) LastHealthProbe(hostname string) (int, bool) {
	utm.mu.RLock()
	defer utm.mu.RUnlock()
	p, ok := utm.lastHealthProbe[hostname]
	if !ok {
		return 0, false
	}
	return p.code, true
}

// ProbeAndRecord GETs the hostname, stores the result, and returns whether
// health checks should treat the tunnel as alive (including Cloudflare 530).
func (utm *UnifiedTunnelManager) ProbeAndRecord(hostname string) bool {
	code, err := ProbeHostname(hostname)
	utm.recordHealthProbe(hostname, code, err)
	if err != nil && code == 0 {
		return false
	}
	return IsHealthCheckHealthy(code)
}
