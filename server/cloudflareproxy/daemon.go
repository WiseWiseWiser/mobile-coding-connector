package cloudflareproxy

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

// ErrDaemonDown is returned when the local-agent keep-alive control socket is absent.
var ErrDaemonDown = errors.New("local agent daemon is not running; start the local-agent macOS app, or use --foreground")

// BindDomainFunc optionally publishes config.domain to cloudflared → :port.
type BindDomainFunc func(domain string, port int) error

// HostOptions configures Host (zero values → ~/.ai-critic/cloudflare-proxy).
type HostOptions struct {
	BaseDir    string
	BindDomain BindDomainFunc
}

// Host runs the control socket for the lifetime of ctx and auto-starts
// the HTTP listener when config autoStart is true.
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
	_ = os.Remove(sockPath)

	h := &host{
		base:       base,
		cfgPath:    ConfigPath(base),
		bindDomain: opts.BindDomain,
	}
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("bind control socket %s: %w", sockPath, err)
	}
	h.ln = ln

	cfg, err := LoadConfig(h.cfgPath)
	if err == nil && cfg.AutoStart {
		if _, _, err := h.startHTTP(cfg); err != nil {
			// boot auto-start is best-effort
			_ = err
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
	base       string
	cfgPath    string
	bindDomain BindDomainFunc
	ln         net.Listener

	mu     sync.Mutex
	srv    *http.Server
	httpLn net.Listener
	proxy  *Server
	port   int
}

func (h *host) cfgLocked() Config {
	cfg, err := LoadConfig(h.cfgPath)
	if err != nil {
		return Config{}
	}
	return cfg
}

func (h *host) startHTTP(cfg Config) (warnings []string, rotated bool, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.srv != nil {
		return nil, false, errors.New("already running")
	}
	rotated, err = prepareAuth(&cfg)
	if err != nil {
		return nil, false, err
	}
	if err := SaveConfig(h.cfgPath, cfg); err != nil {
		return nil, false, err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.port())
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, false, err
	}
	proxy := newServer(cfg, h.base)
	proxy.BindHostname = h.bindDomain
	hs := &http.Server{Handler: proxy}
	h.srv = hs
	h.httpLn = ln
	h.proxy = proxy
	h.port = cfg.port()
	go func() { _ = hs.Serve(ln) }()

	if trim(cfg.Domain) == "" {
		warnings = append(warnings, "warning: domain is empty; not binding a Cloudflare hostname")
	} else if h.bindDomain == nil {
		warnings = append(warnings, "warning: domain is set but Cloudflare bind is not configured")
	} else if berr := h.bindDomain(trim(cfg.Domain), cfg.port()); berr != nil {
		warnings = append(warnings, "warning: Cloudflare bind: "+berr.Error())
	}
	return warnings, rotated, nil
}

func (h *host) stopHTTP() {
	h.mu.Lock()
	srv := h.srv
	h.srv = nil
	h.httpLn = nil
	h.proxy = nil
	h.mu.Unlock()
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(ctx)
		cancel()
		_ = srv.Close()
	}
}

func (h *host) Close() error {
	h.shutdown()
	return nil
}

func (h *host) shutdown() {
	h.stopHTTP()
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
	writeResp(conn, h.apply(req))
}

func (h *host) apply(req OpRequest) OpResponse {
	switch req.Op {
	case "status":
		return OpResponse{OK: true, KV: h.status()}
	case "start":
		cfg := h.cfgLocked()
		applyOverrides(&cfg, req)
		if err := SaveConfig(h.cfgPath, cfg); err != nil {
			return OpResponse{OK: false, Error: err.Error()}
		}
		h.mu.Lock()
		running := h.srv != nil
		h.mu.Unlock()
		if running {
			return OpResponse{OK: true, KV: h.status(), Output: "warning: cloudflare-proxy already running on 127.0.0.1:" + fmt.Sprint(cfg.port())}
		}
		warns, rotated, err := h.startHTTP(cfg)
		if err != nil {
			return OpResponse{OK: false, Error: err.Error()}
		}
		kv := h.status()
		if rotated {
			kv["auth_rotated"] = "true"
		}
		return OpResponse{OK: true, KV: kv, Output: joinWarnings(warns)}
	case "stop":
		h.mu.Lock()
		running := h.srv != nil
		h.mu.Unlock()
		h.stopHTTP()
		kv := h.status()
		if !running {
			return OpResponse{OK: true, KV: kv, Output: "warning: cloudflare-proxy is not running"}
		}
		return OpResponse{OK: true, KV: kv}
	case "enable":
		cfg := h.cfgLocked()
		applyOverrides(&cfg, req)
		cfg.AutoStart = true
		if err := SaveConfig(h.cfgPath, cfg); err != nil {
			return OpResponse{OK: false, Error: err.Error()}
		}
		if req.Now {
			h.mu.Lock()
			running := h.srv != nil
			h.mu.Unlock()
			if !running {
				if _, _, err := h.startHTTP(cfg); err != nil {
					return OpResponse{OK: false, Error: err.Error()}
				}
			}
		}
		return OpResponse{OK: true, KV: h.status()}
	case "disable":
		cfg := h.cfgLocked()
		cfg.AutoStart = false
		if err := SaveConfig(h.cfgPath, cfg); err != nil {
			return OpResponse{OK: false, Error: err.Error()}
		}
		if req.Now {
			h.stopHTTP()
		}
		return OpResponse{OK: true, KV: h.status()}
	default:
		return OpResponse{OK: false, Error: fmt.Sprintf("unknown op %q", req.Op)}
	}
}

func joinWarnings(warns []string) string {
	var out string
	for _, w := range warns {
		w = trim(w)
		if w == "" {
			continue
		}
		if out != "" {
			out += "\n"
		}
		out += w
	}
	return out
}

func (h *host) status() map[string]string {
	cfg := h.cfgLocked()
	kv := map[string]string{
		"listen":    fmt.Sprintf("127.0.0.1:%d", cfg.port()),
		"autoStart": fmt.Sprintf("%v", cfg.AutoStart),
		"auth":      cfg.authSource(),
		"domain":    trim(cfg.Domain),
	}
	if trim(cfg.Domain) == "" {
		kv["domain_bound"] = "no"
	} else if h.bindDomain == nil {
		kv["domain_bound"] = "no"
	} else {
		kv["domain_bound"] = "yes"
	}
	h.mu.Lock()
	running := h.srv != nil
	proxy := h.proxy
	h.mu.Unlock()
	if running {
		kv["status"] = "running"
	} else {
		kv["status"] = "stopped"
	}
	n, conn := 0, 0
	if proxy != nil {
		n, conn = proxy.mappingCount()
	}
	kv["mappings"] = fmt.Sprint(n)
	kv["connected"] = fmt.Sprint(conn)
	return kv
}

func applyOverrides(cfg *Config, req OpRequest) {
	if req.Port > 0 {
		cfg.Port = req.Port
	}
	if req.Domain != "" {
		cfg.Domain = req.Domain
	}
	if req.Token != "" {
		cfg.Token = req.Token
	}
}

// OpRequest is a control-socket command.
type OpRequest struct {
	Op     string `json:"op"`
	Now    bool   `json:"now,omitempty"`
	Port   int    `json:"port,omitempty"`
	Domain string `json:"domain,omitempty"`
	Token  string `json:"token,omitempty"`
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

// ControlClient sends one op to the daemon control socket.
func ControlClient(sockPath string, req OpRequest) (OpResponse, error) {
	conn, err := net.DialTimeout("unix", sockPath, 2*time.Second)
	if err != nil {
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

// RunForeground serves :port in this process until ctx is done.
// onReady is called after listen (and optional CF bind) before Serve blocks.
func RunForeground(ctx context.Context, base string, cfg Config, bind BindDomainFunc, onReady func(warnings []string, cfg Config)) error {
	if err := os.MkdirAll(base, 0o700); err != nil {
		return err
	}
	rotated, err := prepareAuth(&cfg)
	if err != nil {
		return err
	}
	if err := SaveConfig(ConfigPath(base), cfg); err != nil {
		return err
	}
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.port())
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	proxy := newServer(cfg, base)
	proxy.BindHostname = bind
	hs := &http.Server{Handler: proxy}
	var warnings []string
	if trim(cfg.Domain) == "" {
		warnings = append(warnings, "warning: domain is empty; not binding a Cloudflare hostname")
	} else if bind == nil {
		warnings = append(warnings, "warning: domain is set but Cloudflare bind is not configured")
	} else if berr := bind(trim(cfg.Domain), cfg.port()); berr != nil {
		warnings = append(warnings, "warning: Cloudflare bind: "+berr.Error())
	}
	if rotated {
		cfg = mustLoad(ConfigPath(base), cfg)
	}
	if onReady != nil {
		onReady(warnings, cfg)
	}
	go func() {
		<-ctx.Done()
		_ = hs.Close()
	}()
	err = hs.Serve(ln)
	if err == http.ErrServerClosed || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func mustLoad(path string, fallback Config) Config {
	cfg, err := LoadConfig(path)
	if err != nil {
		return fallback
	}
	return cfg
}
