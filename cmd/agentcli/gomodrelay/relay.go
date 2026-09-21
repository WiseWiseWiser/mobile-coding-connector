// Package gomodrelay implements the local fast-fail GOPROXY relay.
//
// The relay listens on 127.0.0.1:21001 and forwards requests to an upstream
// GOPROXY. When the upstream is unreachable (connect timeout, pre-header hang,
// connection reset before any response byte), it responds with 404 so the
// go command's GOPROXY chain falls through to the next entry, and logs a
// warning. A health-state cache keeps the fast-fail latency at ~0 ms while
// the upstream is down: the state flips on first failure and a background
// re-probe flips it back.
//
// The listener is hosted by the always-running local agent daemon
// (`remote-agent ssh --serve`); control happens over a unix socket.
package gomodrelay

import (
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Defaults for the local relay.
const (
	DefaultPort        = 21001
	DefaultDialTimeout = 2 * time.Second
	DefaultDownRecheck = 10 * time.Second
	respHeaderTimeout  = 30 * time.Second
)

// Config is the persisted relay config (daemon-owned).
type Config struct {
	Enabled       bool   `json:"enabled"`
	Port          int    `json:"port,omitempty"`
	Upstream      string `json:"upstream,omitempty"`
	DialTimeoutMS int    `json:"dial_timeout_ms,omitempty"`
	DownRecheckMS int    `json:"down_recheck_ms,omitempty"`
}

func (c Config) dialTimeout() time.Duration {
	if c.DialTimeoutMS > 0 {
		return time.Duration(c.DialTimeoutMS) * time.Millisecond
	}
	return DefaultDialTimeout
}

func (c Config) downRecheck() time.Duration {
	if c.DownRecheckMS > 0 {
		return time.Duration(c.DownRecheckMS) * time.Millisecond
	}
	return DefaultDownRecheck
}

func (c Config) port() int {
	if c.Port > 0 {
		return c.Port
	}
	return DefaultPort
}

// Relay forwards requests to the upstream with fast-fail 404 semantics.
type Relay struct {
	cfg    Config
	logf   func(format string, args ...any)
	client *http.Client

	mu       sync.Mutex
	up       bool
	lastOK   time.Time
	misses   int
	probing  bool
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewRelay creates a relay. logf receives runtime lines (may be nil).
func NewRelay(cfg Config, logf func(format string, args ...any)) *Relay {
	r := &Relay{cfg: cfg, logf: logf, stopCh: make(chan struct{})}
	dialer := &net.Dialer{Timeout: cfg.dialTimeout()}
	r.client = &http.Client{
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			ResponseHeaderTimeout: respHeaderTimeout,
			MaxIdleConnsPerHost:   8,
			IdleConnTimeout:       90 * time.Second,
		},
	}
	return r
}

// ProbeUp dials the upstream host synchronously and seeds the health state.
func (r *Relay) ProbeUp() (bool, time.Duration, error) {
	host := upstreamHost(r.cfg.Upstream)
	t0 := time.Now()
	conn, err := net.DialTimeout("tcp", host, r.cfg.dialTimeout())
	elapsed := time.Since(t0)
	if err != nil {
		r.markDown()
		return false, elapsed, err
	}
	_ = conn.Close()
	r.setUp()
	return true, elapsed, nil
}

func (r *Relay) setUp() {
	r.mu.Lock()
	first := !r.up
	r.up = true
	r.lastOK = time.Now()
	r.mu.Unlock()
	if first && r.logf != nil {
		r.logf("upstream up (%s)", r.cfg.Upstream)
	}
}

func (r *Relay) markDown() {
	r.mu.Lock()
	was := r.up
	r.up = false
	probing := r.probing
	if !probing {
		r.probing = true
	}
	r.mu.Unlock()
	if !was && r.logf != nil {
		// already down; no repeat state line
	}
	if !probing {
		go r.probeLoop()
	}
}

// probeLoop re-probes until the upstream is reachable again.
func (r *Relay) probeLoop() {
	defer func() {
		r.mu.Lock()
		r.probing = false
		r.mu.Unlock()
	}()
	t := time.NewTicker(r.cfg.downRecheck())
	defer t.Stop()
	for {
		select {
		case <-r.stopCh:
			return
		case <-t.C:
			host := upstreamHost(r.cfg.Upstream)
			conn, err := net.DialTimeout("tcp", host, r.cfg.dialTimeout())
			if err != nil {
				continue
			}
			_ = conn.Close()
			r.setUp()
			return
		}
	}
}

// ServeHTTP implements the fast-fail forwarder.
func (r *Relay) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	up := r.up
	r.mu.Unlock()
	if !up {
		r.reject(w, req)
		return
	}
	outURL := upstreamURL(r.cfg.Upstream, req.URL)
	outReq, err := http.NewRequest(req.Method, outURL, req.Body)
	if err != nil {
		r.reject(w, req)
		return
	}
	for _, h := range []string{"Accept", "Accept-Encoding", "Range", "User-Agent"} {
		if v := req.Header.Get(h); v != "" {
			outReq.Header.Set(h, v)
		}
	}
	resp, err := r.client.Do(outReq)
	if err != nil {
		// pre-response failure: connect refused/timeout, pre-header hang, reset
		r.markDown()
		r.reject(w, req)
		return
	}
	r.setUp()
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		if strings.EqualFold(k, "Connection") || strings.EqualFold(k, "Keep-Alive") ||
			strings.EqualFold(k, "Proxy-Connection") || strings.EqualFold(k, "Transfer-Encoding") {
			continue
		}
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (r *Relay) reject(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	r.misses++
	r.mu.Unlock()
	if r.logf != nil {
		r.logf("warning: upstream unreachable, 404 for %s %s (chain falls through)", req.Method, req.URL.Path)
	}
	http.Error(w, "upstream unavailable", http.StatusNotFound)
}

// Close stops background probing. Safe to call more than once.
func (r *Relay) Close() {
	r.stopOnce.Do(func() { close(r.stopCh) })
}

// EffectivePort returns the effective port (cfg or default).
func (c Config) EffectivePort() int { return c.port() }

// EffectiveDownRecheck returns the effective re-check interval.
func (c Config) EffectiveDownRecheck() time.Duration { return c.downRecheck() }
