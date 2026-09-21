package gomod

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Default paths under the server home.
const (
	DefaultPort = 21000
	// DefaultRoot is the on-disk GOMODCACHE cache/download tree to serve.
	DefaultRoot = "/root/gomod-proxy/cache/download"
	// LogFile and ConfigFile live under $HOME/.ai-critic/go/.
	LogFileName    = "mod-proxy.log"
	ConfigFileName = "mod-proxy.json"
)

// Config is the persisted mod-proxy service config.
type Config struct {
	Enabled bool   `json:"enabled"`
	Port    int    `json:"port,omitempty"`
	Root    string `json:"root,omitempty"`
}

// Manager owns the in-process mod-proxy listener.
type Manager struct {
	mu        sync.Mutex
	cfg       Config
	srv       *http.Server
	startedAt time.Time
	logPath   string
}

var (
	defaultMu sync.Mutex
	defaultM  *Manager
)

// DefaultManager returns the process-wide manager (shared by the HTTP API
// and BootAutoStart so Start/Status observe the same listener).
func DefaultManager() (*Manager, error) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultM != nil {
		return defaultM, nil
	}
	m, err := NewManager()
	if err != nil {
		return nil, err
	}
	defaultM = m
	return m, nil
}

// NewManager creates a manager with default paths.
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".ai-critic", "go")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	return &Manager{
		logPath: filepath.Join(dir, LogFileName),
	}, nil
}

func (m *Manager) configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ai-critic", "go", ConfigFileName)
}

func (m *Manager) loadConfig() (Config, error) {
	cfg := Config{Port: DefaultPort, Root: DefaultRoot}
	data, err := os.ReadFile(m.configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	var saved Config
	if err := json.Unmarshal(data, &saved); err != nil {
		return cfg, fmt.Errorf("corrupt %s: %w", m.configPath(), err)
	}
	if saved.Port > 0 {
		cfg.Port = saved.Port
	}
	if saved.Root != "" {
		cfg.Root = saved.Root
	}
	cfg.Enabled = saved.Enabled
	return cfg, nil
}

func (m *Manager) saveConfig(cfg Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	path := m.configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (m *Manager) logf(format string, args ...any) {
	f, err := os.OpenFile(m.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, time.Now().Format("2006-01-02 15:04:05")+" "+format+"\n", args...)
}

// Start binds the listener and serves the proxy. Idempotent.
func (m *Manager) Start() (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.srv != nil {
		return m.statusLocked(), nil
	}
	cfg, err := m.loadConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Root == "" {
		cfg.Root = DefaultRoot
	}
	if !rootHasModules(cfg.Root) {
		return nil, fmt.Errorf("proxy root %s does not exist or has no modules", cfg.Root)
	}
	addr := fmt.Sprintf(":%d", cfg.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	m.cfg = cfg
	m.startedAt = time.Now()
	m.srv = &http.Server{Handler: NewProxyHandler(cfg.Root)}
	go func() {
		_ = m.srv.Serve(ln)
	}()
	m.logf("started on %s root=%s", addr, cfg.Root)
	kv := m.statusLocked()
	return kv, nil
}

// Stop shuts the listener down. Idempotent.
func (m *Manager) Stop() (map[string]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.srv == nil {
		return nil, nil // caller prints warning
	}
	srv := m.srv
	m.srv = nil
	m.logf("stopped")
	_ = srv.Close()
	return map[string]string{"status": "stopped"}, nil
}

// Enable marks the service for auto-start on agent server boot.
func (m *Manager) Enable(now bool) (map[string]string, error) {
	m.mu.Lock()
	cfg, err := m.loadConfig()
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	cfg.Enabled = true
	err = m.saveConfig(cfg)
	m.mu.Unlock()
	if err != nil {
		return nil, err
	}
	m.logf("enabled")
	if now {
		return m.Start()
	}
	return map[string]string{"auto_start": "enabled"}, nil
}

// Disable clears auto-start-on-boot. A running listener is untouched.
func (m *Manager) Disable() (map[string]string, error) {
	m.mu.Lock()
	cfg, err := m.loadConfig()
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	cfg.Enabled = false
	err = m.saveConfig(cfg)
	m.mu.Unlock()
	if err != nil {
		return nil, err
	}
	m.logf("disabled")
	return map[string]string{"auto_start": "disabled"}, nil
}

// Status reports runtime + config state.
func (m *Manager) Status() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statusLocked()
}

func (m *Manager) statusLocked() map[string]string {
	kv := map[string]string{}
	if m.srv == nil {
		kv["status"] = "stopped"
	} else {
		kv["status"] = "running"
		kv["uptime"] = time.Since(m.startedAt).Round(time.Second).String()
		kv["port"] = fmt.Sprintf("%d", m.cfg.Port)
		kv["root"] = m.cfg.Root
		kv["modules"] = fmt.Sprintf("%d", CountModules(m.cfg.Root))
	}
	// config view (always present, even when stopped)
	saved, err := m.loadConfig()
	if err == nil {
		if saved.Enabled {
			kv["auto_start"] = "enabled"
		} else {
			kv["auto_start"] = "disabled"
		}
		if m.srv == nil {
			kv["port"] = fmt.Sprintf("%d", saved.Port)
			kv["root"] = saved.Root
		}
	}
	return kv
}

// SetRoot persists a non-default data root (used by start --root).
func (m *Manager) SetRoot(root string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg, err := m.loadConfig()
	if err != nil {
		return err
	}
	cfg.Root = root
	return m.saveConfig(cfg)
}

// SetPort persists a non-default port.
func (m *Manager) SetPort(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cfg, err := m.loadConfig()
	if err != nil {
		return err
	}
	cfg.Port = port
	return m.saveConfig(cfg)
}

// BootAutoStart binds the listener at agent-server boot when enabled.
// Never fatal: failures are logged and swallowed.
func (m *Manager) BootAutoStart() {
	cfg, err := m.loadConfig()
	if err != nil || !cfg.Enabled {
		return
	}
	if _, err := m.Start(); err != nil {
		m.logf("boot auto-start failed: %v", err)
	}
}

// TailLog returns the last n lines of the service log (n<=0 -> all).
func (m *Manager) TailLog(n int) (string, error) {
	f, err := os.Open(m.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out, nil
}

func rootHasModules(root string) bool {
	entries, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			return true
		}
	}
	return false
}

// BootAutoStart binds the listener at agent-server boot when the persisted
// config has enabled=true. Package-level entry used by server/server.go.
// Never fatal.
func BootAutoStart() {
	m, err := DefaultManager()
	if err != nil {
		return
	}
	m.BootAutoStart()
}
