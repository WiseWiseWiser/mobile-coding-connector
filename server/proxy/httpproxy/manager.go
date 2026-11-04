package httpproxy

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/ai-critic/server/proxy/portforward"
)

// Manager owns the proxy listener, its config, the published domain mapping
// and the live stats. One proxy per server; use GetManager().
type Manager struct {
	mu sync.Mutex

	cfg           *Config
	srv           *http.Server
	ln            net.Listener
	port          int
	stats         *Stats
	handler       *proxyHandler
	running       bool
	startedAt     time.Time
	publishErr    error
	publishedPort int
}

var (
	defaultManagerOnce sync.Once
	defaultManager     *Manager
)

func GetManager() *Manager {
	defaultManagerOnce.Do(func() {
		defaultManager = &Manager{stats: NewStats()}
	})
	return defaultManager
}

// currentConfig returns the live config snapshot (safe copy).
func (m *Manager) currentConfig() *Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cfg == nil {
		return defaultConfig()
	}
	cp := *m.cfg
	return &cp
}

// transport is shared by all forwarded plain-HTTP requests.
var transport = &http.Transport{
	MaxIdleConns:        64,
	IdleConnTimeout:     90 * time.Second,
	TLSHandshakeTimeout: 10 * time.Second,
}

// Start binds the listener and, when a Domain is configured, publishes it
// through the cloudflare_owned port-forward provider (same stack user
// services use). Starting an already-running proxy is an error; stop first.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return fmt.Errorf("http proxy is already running")
	}
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	cfg.normalized()
	if cfg.Password == "" {
		return fmt.Errorf("http proxy has no password configured; set one before starting")
	}
	m.cfg = cfg
	m.handler = newProxyHandler(m)
	m.stats = NewStats()

	ln, err := net.Listen("tcp", net.JoinHostPort(cfg.BindAddr, fmt.Sprintf("%d", cfg.ListenPort)))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	m.ln = ln
	m.port = ln.Addr().(*net.TCPAddr).Port
	m.srv = &http.Server{Handler: m.handler, ReadHeaderTimeout: 30 * time.Second}
	m.running = true
	m.startedAt = time.Now()
	go func() {
		_ = m.srv.Serve(ln)
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	// LAN/direct access is up once Serve started; the domain mapping is
	// best-effort and its failure is visible in Status.
	if err := m.publishLocked(cfg); err != nil {
		if cfg.Domain != "" {
			fmt.Printf("[http-proxy] domain publish failed: %v\n", err)
			m.publishErr = err
		}
	} else {
		m.publishErr = nil
	}
	return nil
}

// publishLocked maps the configured owned domain onto the proxy port.
func (m *Manager) publishLocked(cfg *Config) error {
	if cfg.Domain == "" {
		return nil
	}
	pfm := portforward.GetDefaultManager()
	if _, err := pfm.Add(m.port, cfg.Domain, portforward.ProviderCloudflareOwned); err != nil {
		return err
	}
	m.publishedPort = m.port
	return nil
}

// unpublishLocked drops the domain mapping, if we published one.
func (m *Manager) unpublishLocked() {
	if m.publishedPort <= 0 {
		return
	}
	pfm := portforward.GetDefaultManager()
	_ = pfm.RemoveSelector(config.LocalMappingID(m.publishedPort))
	m.publishedPort = 0
}

// Stop shuts the listener down and removes the published mapping.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unpublishLocked()
	if !m.running {
		return nil
	}
	srv := m.srv
	ln := m.ln
	m.running = false
	if srv != nil {
		go func() { _ = srv.Close() }()
	}
	if ln != nil {
		_ = ln.Close()
	}
	return nil
}

// Status is the live state for the services manager and the detail page.
type Status struct {
	Running    bool   `json:"running"`
	Port       int    `json:"port,omitempty"`
	PublicURL  string `json:"public_url,omitempty"`
	Detail     string `json:"detail"`
	AuthUser   string `json:"auth_user"`
	Domain     string `json:"domain,omitempty"`
	PublishErr string `json:"publish_error,omitempty"`
	AutoStart  bool   `json:"auto_start"`
	UptimeSecs int64  `json:"uptime_secs,omitempty"`
	Configured bool   `json:"configured"`
}

// Status reports the live state.
func (m *Manager) Status() *Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg := m.cfg
	out := &Status{Configured: ConfigExists()}
	if cfg != nil {
		out.AuthUser = cfg.Username
		out.Domain = cfg.Domain
		out.AutoStart = cfg.AutoStart
	}
	if m.running {
		out.Running = true
		out.Port = m.port
		out.Detail = fmt.Sprintf("listening :%d", m.port)
		out.UptimeSecs = int64(time.Since(m.startedAt).Seconds())
		if m.publishErr != nil {
			out.PublishErr = m.publishErr.Error()
		}
	} else {
		out.Detail = "not running"
	}
	if out.PublicURL == "" && out.Domain != "" {
		out.PublicURL = "https://" + out.Domain
	}
	return out
}

// RotatePassword generates a new random password, persists it and returns it.
// Active connections are unaffected; new ones must use the new credential.
func (m *Manager) RotatePassword() (string, error) {
	const alphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	pw := make([]byte, 16)
	for i := range pw {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		pw[i] = alphabet[n.Int64()]
	}
	m.mu.Lock()
	cfg := m.cfg
	m.mu.Unlock()
	if cfg == nil {
		loaded, err := LoadConfig()
		if err != nil {
			return "", err
		}
		cfg = loaded
	}
	cfg.Password = string(pw)
	if err := SaveConfig(cfg); err != nil {
		return "", err
	}
	m.mu.Lock()
	m.cfg = cfg
	m.mu.Unlock()
	return cfg.Password, nil
}

// Connections renders the live + recent connection list.
func (m *Manager) Connections() (items []ConnInfo, active, connectActive, total, failures int64) {
	return m.stats.Snapshot()
}

// BootAutoStart binds the listener at agent-server boot when enabled.
func BootAutoStart() {
	cfg, err := LoadConfig()
	if err != nil || !cfg.AutoStart {
		return
	}
	if cfg.Password == "" {
		fmt.Printf("[http-proxy] autostart skipped: no password configured\n")
		return
	}
	if err := GetManager().Start(); err != nil {
		fmt.Printf("[http-proxy] autostart failed: %v\n", err)
	}
}
