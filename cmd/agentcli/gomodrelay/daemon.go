package gomodrelay

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// ErrDaemonDown is returned by CLI clients when the local agent daemon
// (remote-agent ssh --serve) is not running, so the control socket is absent.
var ErrDaemonDown = errors.New("local agent daemon is not running; start the local-agent macOS app (ai-critic-macos)")

// HostOptions configures Host (zero values -> defaults under ~/.ai-critic/go).
type HostOptions struct {
	// BaseDir overrides ~/.ai-critic/go (tests).
	BaseDir string
	// Stderr receives relay warnings when the daemon wants them on its console.
	Stderr *os.File
}

// Host runs the relay control endpoint for the lifetime of ctx and
// auto-starts the relay when the persisted config has Enabled=true.
// It is hosted inside the always-running local agent daemon.
func Host(ctx context.Context, opts HostOptions) (ioCloser, error) {
	base := opts.BaseDir
	if base == "" {
		var err error
		base, err = BaseDir()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}
	sockPath := SocketPath(base)
	_ = os.Remove(sockPath) // stale socket from a previous daemon run

	h := &host{
		base:    base,
		cfgPath: ConfigPath(base),
		logw:    newLogWriter(LogPath(base)),
	}
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("bind control socket %s: %w", sockPath, err)
	}
	h.ln = ln

	cfg, err := LoadConfig(h.cfgPath)
	if err == nil && cfg.Enabled {
		if err := h.startRelay(cfg); err != nil {
			h.logw.logf("boot auto-start failed: %v", err)
		} else {
			h.logw.logf("boot auto-start: relay listening on 127.0.0.1:%d -> %s", h.relayCfg().port(), h.relayCfg().Upstream)
		}
	}

	go func() {
		<-ctx.Done()
		h.shutdown()
	}()
	go h.acceptLoop()
	return h, nil
}

type ioCloser interface {
	Close() error
}

type host struct {
	base    string
	cfgPath string
	logw    *logWriter
	ln      net.Listener

	mu    sync.Mutex
	relay *Relay
	srv   *http.Server
	port  int
}

func (h *host) relayCfg() Config {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.cfgLocked()
}

func (h *host) cfgLocked() Config {
	cfg, err := LoadConfig(h.cfgPath)
	if err != nil {
		return Config{}
	}
	return cfg
}

func (h *host) startRelay(cfg Config) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.relay != nil {
		return errors.New("relay already running")
	}
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.port())
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	relay := NewRelay(cfg, h.logw.logf)
	h.relay = relay
	h.port = cfg.port()
	h.srv = &http.Server{Handler: relay}
	go func() {
		_ = h.srv.Serve(ln)
	}()
	h.mu.Unlock()
	if ok, elapsed, err := relay.ProbeUp(); !ok && err != nil {
		h.logw.logf("upstream %s unreachable (dial %s: %v); marked down, re-checking every %s",
			cfg.Upstream, elapsed.Round(time.Millisecond), err, cfg.downRecheck())
	} else if ok {
		h.logw.logf("upstream %s up (probed in %s)", cfg.Upstream, elapsed.Round(time.Millisecond))
	}
	h.mu.Lock()
	return nil
}

func (h *host) stopRelay() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.relay == nil {
		return nil
	}
	relay := h.relay
	srv := h.srv
	h.relay = nil
	h.srv = nil
	relay.Close()
	if srv != nil {
		_ = srv.Close()
	}
	h.logw.logf("relay stopped")
	return nil
}

func (h *host) Close() error {
	h.shutdown()
	return nil
}

func (h *host) shutdown() {
	_ = h.stopRelay()
	_ = h.ln.Close()
	_ = os.Remove(SocketPath(h.base))
}

func (h *host) acceptLoop() {
	for {
		conn, err := h.ln.Accept()
		if err != nil {
			return
		}
		go h.handleConn(conn)
	}
}

func (h *host) handleConn(conn net.Conn) {
	defer conn.Close()
	var req OpRequest
	dec := json.NewDecoder(conn)
	if err := dec.Decode(&req); err != nil {
		writeResp(conn, OpResponse{OK: false, Error: "invalid request: " + err.Error()})
		return
	}
	resp := h.apply(req)
	writeResp(conn, resp)
}

func (h *host) apply(req OpRequest) OpResponse {
	switch req.Op {
	case "status":
		return OpResponse{OK: true, KV: h.status()}
	case "start":
		cfg := h.cfgLocked()
		applyOverrides(&cfg, req)
		if cfg.Upstream == "" {
			return OpResponse{OK: false, Error: "upstream is required (e.g. --upstream http://10.91.186.143:21000)"}
		}
		if err := SaveConfig(h.cfgPath, cfg); err != nil {
			return OpResponse{OK: false, Error: err.Error()}
		}
		h.mu.Lock()
		running := h.relay != nil
		h.mu.Unlock()
		if running {
			return OpResponse{OK: true, KV: h.status(), Output: "warning: mod-proxy-relay already running on 127.0.0.1:" + fmt.Sprint(cfg.port())}
		}
		if err := h.startRelay(cfg); err != nil {
			return OpResponse{OK: false, Error: err.Error()}
		}
		return OpResponse{OK: true, KV: h.status()}
	case "stop":
		cfg := h.cfgLocked()
		_ = h.stopRelay()
		kv := map[string]string{"status": "stopped", "auto_start": autoStartStr(cfg.Enabled), "port": fmt.Sprint(cfg.port()), "upstream": cfg.Upstream}
		return OpResponse{OK: true, KV: kv}
	case "enable":
		cfg := h.cfgLocked()
		applyOverrides(&cfg, req)
		if cfg.Upstream == "" {
			return OpResponse{OK: false, Error: "upstream is required (e.g. --upstream http://10.91.186.143:21000)"}
		}
		cfg.Enabled = true
		if err := SaveConfig(h.cfgPath, cfg); err != nil {
			return OpResponse{OK: false, Error: err.Error()}
		}
		if req.Now {
			h.mu.Lock()
			running := h.relay != nil
			h.mu.Unlock()
			if !running {
				if err := h.startRelay(cfg); err != nil {
					return OpResponse{OK: false, Error: err.Error()}
				}
			}
		}
		return OpResponse{OK: true, KV: h.status()}
	case "disable":
		cfg := h.cfgLocked()
		cfg.Enabled = false
		if err := SaveConfig(h.cfgPath, cfg); err != nil {
			return OpResponse{OK: false, Error: err.Error()}
		}
		if req.Now {
			_ = h.stopRelay()
		}
		return OpResponse{OK: true, KV: h.status()}
	default:
		return OpResponse{OK: false, Error: fmt.Sprintf("unknown op %q", req.Op)}
	}
}

func (h *host) status() map[string]string {
	cfg := h.cfgLocked()
	kv := map[string]string{
		"auto_start": autoStartStr(cfg.Enabled),
		"port":       fmt.Sprint(cfg.port()),
		"upstream":   cfg.Upstream,
	}
	h.mu.Lock()
	relay := h.relay
	h.mu.Unlock()
	if relay == nil {
		kv["status"] = "stopped"
		return kv
	}
	kv["status"] = "running"
	relay.mu.Lock()
	up := relay.up
	misses := relay.misses
	lastOK := relay.lastOK
	relay.mu.Unlock()
	if up {
		kv["health"] = "up"
	} else {
		kv["health"] = "down"
	}
	if !lastOK.IsZero() {
		kv["last_ok"] = time.Since(lastOK).Round(time.Second).String() + " ago"
	}
	kv["misses"] = fmt.Sprint(misses)
	return kv
}

func autoStartStr(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}

func applyOverrides(cfg *Config, req OpRequest) {
	if req.Port > 0 {
		cfg.Port = req.Port
	}
	if req.Upstream != "" {
		cfg.Upstream = req.Upstream
	}
	if req.DialTimeoutMS > 0 {
		cfg.DialTimeoutMS = req.DialTimeoutMS
	}
	if req.DownRecheckMS > 0 {
		cfg.DownRecheckMS = req.DownRecheckMS
	}
}

// OpRequest is a control-socket command.
type OpRequest struct {
	Op            string `json:"op"`
	Now           bool   `json:"now,omitempty"`
	Port          int    `json:"port,omitempty"`
	Upstream      string `json:"upstream,omitempty"`
	DialTimeoutMS int    `json:"dial_timeout_ms,omitempty"`
	DownRecheckMS int    `json:"down_recheck_ms,omitempty"`
	Lines         int    `json:"lines,omitempty"`
}

// OpResponse is a control-socket reply.
type OpResponse struct {
	OK     bool              `json:"ok"`
	KV     map[string]string `json:"kv,omitempty"`
	Output string            `json:"output,omitempty"`
	Error  string            `json:"error,omitempty"`
}

func writeResp(conn net.Conn, resp OpResponse) {
	data, _ := json.Marshal(resp)
	_, _ = conn.Write(append(data, '\n'))
}

// ControlClient sends one op to the daemon control socket and returns the reply.
func ControlClient(sockPath string, req OpRequest) (OpResponse, error) {
	conn, err := net.DialTimeout("unix", sockPath, 2*time.Second)
	if err != nil {
		// Missing socket, connection refused, or timeout: the daemon is not
		// hosting the control endpoint (not running, or too old).
		return OpResponse{}, ErrDaemonDown
	}
	defer conn.Close()
	data, err := json.Marshal(req)
	if err != nil {
		return OpResponse{}, err
	}
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return OpResponse{}, err
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return OpResponse{}, err
	}
	var resp OpResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return OpResponse{}, err
	}
	return resp, nil
}
