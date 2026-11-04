package httpproxy

import (
	"crypto/subtle"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"
)

// hopHeaders are hop-by-hop per RFC 7230 §6.1 and must not cross a proxy hop.
var hopHeaders = []string{
	"Proxy-Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
	"Te", "Trailer", "Transfer-Encoding", "Upgrade",
}

// proxyHandler implements the forward proxy. ServeHTTP sees proxy-style
// requests: CONNECT host:port for tunnels, absolute-form GET/POST for plain
// HTTP, and (leniently) origin-form requests whose Host names the target.
type proxyHandler struct {
	mgr *Manager

	failMu   sync.Mutex
	failures map[string][]time.Time // client IP -> failed auth timestamps
}

func newProxyHandler(mgr *Manager) *proxyHandler {
	return &proxyHandler{mgr: mgr, failures: make(map[string][]time.Time)}
}

func (h *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r.RemoteAddr)

	// IP lockout applies before credential evaluation: a brute-forcing client
	// stays locked for the window even if it later presents good credentials.
	cfg := h.mgr.currentConfig()
	if h.locked(ip, cfg) {
		w.Header().Set("Retry-After", "30")
		http.Error(w, "too many failed auth attempts", http.StatusTooManyRequests)
		return
	}

	// Mandatory Basic auth: Proxy-Authorization first, Authorization as the
	// lenient fallback some clients send when absolute-form is used.
	user, pass, ok := r.BasicAuth()
	if !ok {
		user, pass, ok = parseProxyAuth(r)
	}
	authOK := ok &&
		subtle.ConstantTimeCompare([]byte(user), []byte(cfg.Username)) == 1 &&
		subtle.ConstantTimeCompare([]byte(pass), []byte(cfg.Password)) == 1

	if authOK {
		h.clearFailures(ip)
	} else {
		h.noteFailure(ip, cfg)
		h.mgr.stats.CountFailures()
		w.Header().Set("Proxy-Authenticate", `Basic realm="ai-critic http proxy"`)
		http.Error(w, "407 Proxy Authentication Required", http.StatusProxyAuthRequired)
		return
	}

	switch {
	case r.Method == http.MethodConnect:
		h.handleConnect(w, r)
	case r.URL.Host != "" || r.Host != "":
		h.handleHTTP(w, r)
	default:
		http.Error(w, "missing proxy target", http.StatusBadRequest)
	}
}

// targetHostPort resolves where a plain-HTTP proxy request should go.
func (h *proxyHandler) targetHostPort(r *http.Request) string {
	if r.URL.Host != "" {
		return r.URL.Host
	}
	return r.Host
}

// handleHTTP forwards absolute-form (and lenient origin-form) requests.
func (h *proxyHandler) handleHTTP(w http.ResponseWriter, r *http.Request) {
	target := h.targetHostPort(r)
	if target == "" {
		http.Error(w, "missing proxy target", http.StatusBadRequest)
		return
	}
	id, finish := h.mgr.stats.Open(clientIP(r.RemoteAddr), target, r.Method)
	var once sync.Once
	done := func(state, errMsg string) { once.Do(func() { finish(state, errMsg) }) }
	defer done("closed", "")

	scheme := "http"
	if r.URL.Scheme == "https" {
		scheme = "https"
	}
	out := r.Clone(r.Context())
	out.RequestURI = ""
	out.URL.Scheme = scheme
	out.URL.Host = target
	out.Host = target
	for _, hdr := range hopHeaders {
		out.Header.Del(hdr)
	}

	rec := &countingResponseWriter{ResponseWriter: w, stats: h.mgr.stats, id: id}
	proxy := &httputil.ReverseProxy{
		Director:  func(req *http.Request) { /* URL already rewritten above */ },
		Transport: transport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			done("error", err.Error())
			http.Error(w, "proxy error: "+err.Error(), http.StatusBadGateway)
		},
	}
	proxy.ServeHTTP(rec, out)
}

// handleConnect tunnels CONNECT host:port: after auth the proxy dials the
// requested endpoint, answers "200 Connection established", and pipes raw
// bytes both ways. The tunnel carries TLS from the client to the target; the
// proxy never sees plaintext.
func (h *proxyHandler) handleConnect(w http.ResponseWriter, r *http.Request) {
	target := r.Host
	if target == "" {
		target = r.RequestURI
	}
	if target == "" || !strings.Contains(target, ":") {
		http.Error(w, "CONNECT requires host:port", http.StatusBadRequest)
		return
	}
	// Refuse loopback targets so an outside client cannot hop into the proxy
	// host itself through the tunnel. Tests (and loopback-only deployments)
	// may opt out via config.
	cfg := h.mgr.currentConfig()
	if !cfg.AllowLoopbackConnect && isLoopbackTarget(target) {
		http.Error(w, "CONNECT to loopback is not allowed", http.StatusForbidden)
		return
	}

	id, finish := h.mgr.stats.Open(clientIP(r.RemoteAddr), target, "CONNECT")
	var once sync.Once
	done := func(state, errMsg string) { once.Do(func() { finish(state, errMsg) }) }

	backend, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		done("error", "dial "+err.Error())
		http.Error(w, "dial failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer backend.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		done("error", "hijack unsupported")
		http.Error(w, "hijack unsupported", http.StatusInternalServerError)
		return
	}
	client, buf, err := hijacker.Hijack()
	if err != nil {
		done("error", "hijack "+err.Error())
		return
	}
	defer client.Close()

	// 200 line written raw: the connection is no longer HTTP after this.
	if _, err := client.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n")); err != nil {
		done("error", "write 200 "+err.Error())
		return
	}

	go func() {
		_, _ = io.Copy(&byteCounter{w: backend, stats: h.mgr.stats, id: id, up: true}, buf)
		done("closed", "")
		if c, ok := backend.(*net.TCPConn); ok {
			_ = c.CloseWrite()
		}
	}()
	_, _ = io.Copy(&byteCounter{w: client, stats: h.mgr.stats, id: id}, backend)
	done("closed", "")
}

// noteFailure records a failed auth for the per-client lockout window.
func (h *proxyHandler) noteFailure(ip string, cfg *Config) {
	if ip == "" {
		return
	}
	h.failMu.Lock()
	defer h.failMu.Unlock()
	now := time.Now()
	window := time.Duration(cfg.FailWindowSecs) * time.Second
	keep := h.failures[ip][:0]
	for _, t := range h.failures[ip] {
		if now.Sub(t) < window {
			keep = append(keep, t)
		}
	}
	h.failures[ip] = append(keep, now)
}

// clearFailures resets the failure count after a successful auth.
func (h *proxyHandler) clearFailures(ip string) {
	h.failMu.Lock()
	defer h.failMu.Unlock()
	delete(h.failures, ip)
}

// locked reports whether ip exceeded the failure limit inside the window.
func (h *proxyHandler) locked(ip string, cfg *Config) bool {
	window := time.Duration(cfg.FailWindowSecs) * time.Second
	h.failMu.Lock()
	defer h.failMu.Unlock()
	count := 0
	for _, t := range h.failures[ip] {
		if time.Since(t) < window {
			count++
		}
	}
	return count >= cfg.FailLimit
}

// byteCounter counts bytes flowing through an io.Copy in either direction.
type byteCounter struct {
	w     io.Writer
	stats *Stats
	id    uint64
	up    bool
}

func (c *byteCounter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	if n > 0 {
		if c.up {
			c.stats.AddBytes(c.id, int64(n), 0)
		} else {
			c.stats.AddBytes(c.id, 0, int64(n))
		}
	}
	return n, err
}

// countingResponseWriter counts downstream bytes for plain HTTP forwarding.
type countingResponseWriter struct {
	http.ResponseWriter
	stats *Stats
	id    uint64
}

func (c *countingResponseWriter) Write(p []byte) (int, error) {
	n, err := c.ResponseWriter.Write(p)
	if n > 0 {
		c.stats.AddBytes(c.id, 0, int64(n))
	}
	return n, err
}

// parseProxyAuth extracts Basic credentials from Proxy-Authorization.
func parseProxyAuth(r *http.Request) (string, string, bool) {
	raw := r.Header.Get("Proxy-Authorization")
	const prefix = "Basic "
	if len(raw) <= len(prefix) || !strings.EqualFold(raw[:len(prefix)], prefix) {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(raw[len(prefix):])
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func clientIP(remote string) string {
	if host, _, err := net.SplitHostPort(remote); err == nil {
		return host
	}
	return remote
}

func isLoopbackTarget(hostPort string) bool {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}
