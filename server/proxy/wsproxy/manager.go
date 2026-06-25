package wsproxy

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/agent-pro/agent/streaming/sse"
	cfutils "github.com/xhd2015/ai-critic/server/cloudflare"
	"github.com/xhd2015/ai-critic/server/cloudflare/unified_tunnel"
	serverconfig "github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/ai-critic/server/subprocess"
)

const (
	xrayProcID    = "ws-proxy-xray"
	cfQuickProcID = "ws-proxy-cf-quick"
	mappingID     = "ws-proxy"
)

type Status struct {
	Running   bool   `json:"running"`
	PublicURL string `json:"public_url,omitempty"`
	Port      int    `json:"port"`
	IsTmp     bool   `json:"is_tmp"`
}

type VMessConfig struct {
	Host    string `json:"host"`
	Port    string `json:"port"`
	UUID    string `json:"uuid"`
	AlterID string `json:"alter_id"`
	Network string `json:"network"`
	Type    string `json:"type"`
	Path    string `json:"path"`
	TLS     string `json:"tls"`
}

type DryRunResult struct {
	Tmp           bool     `json:"tmp"`
	UpstreamProxy string   `json:"upstream_proxy"`
	ListenPort    int      `json:"listen_port"`
	WSPath        string   `json:"ws_path"`
	Subdomain     string   `json:"subdomain"`
	InstanceID    string   `json:"instance_id"`
	Domain        string   `json:"domain,omitempty"`
	DomainSource  string   `json:"domain_source,omitempty"`
	PublicURL     string   `json:"public_url,omitempty"`
	XrayExists    bool     `json:"xray_exists"`
	XrayPath      string   `json:"xray_path"`
	PortFree      bool     `json:"port_free"`
	Checks        []string `json:"checks"`
	Issues        []string `json:"issues,omitempty"`
}

type xrayClient struct {
	ID      string `json:"id"`
	AlterID int    `json:"alterId"`
}

type xrayWSSettings struct {
	Path string `json:"path"`
}

type xrayStreamSettings struct {
	Network     string         `json:"network"`
	WSSettings  xrayWSSettings `json:"wsSettings"`
}

type xrayInboundSettings struct {
	Clients []xrayClient `json:"clients"`
}

type xrayProxyServer struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type xrayOutboundSettings struct {
	Servers []xrayProxyServer `json:"servers"`
}

type xrayInbound struct {
	Port           int                 `json:"port"`
	Protocol       string              `json:"protocol"`
	Settings       xrayInboundSettings `json:"settings"`
	StreamSettings xrayStreamSettings  `json:"streamSettings"`
}

type xrayOutbound struct {
	Protocol string               `json:"protocol"`
	Settings xrayOutboundSettings `json:"settings"`
}

type xrayConfig struct {
	Inbounds  []xrayInbound  `json:"inbounds"`
	Outbounds []xrayOutbound `json:"outbounds"`
}

var (
	inst     *Manager
	instOnce sync.Once
)

type Manager struct {
	mu        sync.Mutex
	publicURL string
	isTmp     bool
}

func GetManager() *Manager {
	instOnce.Do(func() {
		inst = &Manager{}
	})
	return inst
}

func (m *Manager) Start(tmp bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dr, err := m.dryRunLocked(tmp)
	if err != nil {
		return err
	}

	if len(dr.Issues) > 0 {
		for _, issue := range dr.Issues {
			fmt.Printf("[ws-proxy] Warning: %s\n", issue)
		}
	}

	return m.startLocked(tmp)
}

func (m *Manager) StartStream(w http.ResponseWriter, tmp bool) error {
	sw := sse.NewWriter(w)
	if sw == nil {
		return fmt.Errorf("streaming not supported")
	}

	dr, err := m.DryRun(tmp)
	if err != nil {
		sw.SendError(err.Error())
		return nil
	}
	for _, issue := range dr.Issues {
		sw.SendError(issue)
		return nil
	}

	sw.SendLog("Checks passed, starting ws-proxy...")

	err = m.startLockedStreaming(tmp, sw)
	if err != nil {
		sw.SendError(err.Error())
		return nil
	}

	status := m.Status()
	sw.SendDone(map[string]string{
		"public_url": status.PublicURL,
		"vmess_link": m.GetVMessLink(),
	})
	return nil
}

func (m *Manager) DryRun(tmp bool) (*DryRunResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dryRunLocked(tmp)
}

func (m *Manager) dryRunLocked(tmp bool) (*DryRunResult, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	listenPort := resolvePort(cfg)

	dr := &DryRunResult{
		Tmp:           tmp,
		UpstreamProxy: cfg.UpstreamProxy,
		ListenPort:    listenPort,
		WSPath:        cfg.WSPath,
		Subdomain:     cfg.Subdomain,
		InstanceID:    resolveInstanceID(cfg),
		XrayPath:      xrayBinaryPath(),
	}

	if cfg.UpstreamProxy == "" {
		dr.Checks = append(dr.Checks, "upstream_proxy: NOT SET")
		dr.Issues = append(dr.Issues, "upstream_proxy is not configured")
	} else {
		dr.Checks = append(dr.Checks, fmt.Sprintf("upstream_proxy: %s", cfg.UpstreamProxy))
	}

	if _, err := os.Stat(xrayBinaryPath()); err == nil {
		dr.XrayExists = true
		dr.Checks = append(dr.Checks, "xray binary: exists")
	} else {
		dr.Checks = append(dr.Checks, "xray binary: would be downloaded")
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", listenPort), 500*time.Millisecond)
	if err == nil {
		conn.Close()
		if isXrayAlive(listenPort, cfg.WSPath) {
			dr.PortFree = true
			dr.Checks = append(dr.Checks, fmt.Sprintf("port %d: in use by xray (will reuse)", listenPort))
		} else {
			dr.PortFree = false
			dr.Checks = append(dr.Checks, fmt.Sprintf("port %d: IN USE (not xray)", listenPort))
			dr.Issues = append(dr.Issues, fmt.Sprintf("port %d is already in use by another process", listenPort))
		}
	} else {
		dr.PortFree = true
		if cfg.ListenPort == 0 {
			dr.Checks = append(dr.Checks, fmt.Sprintf("port %d: free (auto-assigned from range %d-%d)", listenPort, portRangeLow, portRangeHigh))
		} else {
			dr.Checks = append(dr.Checks, fmt.Sprintf("port %d: free", listenPort))
		}
	}

	if mgr := subprocess.GetManager(); mgr.IsRunning(xrayProcID) {
		dr.Checks = append(dr.Checks, "xray process: already running")
		dr.Issues = append(dr.Issues, "ws-proxy is already running")
	} else {
		dr.Checks = append(dr.Checks, "xray process: not running (would start)")
	}

	if tmp {
		dr.Checks = append(dr.Checks, "tunnel: Quick Tunnel (trycloudflare.com)")
		dr.PublicURL = "(trycloudflare.com — resolved at runtime)"
	} else {
		dr.Checks = append(dr.Checks, "tunnel: permanent (user domain)")
		domain, source, err := resolveDomainWithSource()
		if err != nil {
			dr.Checks = append(dr.Checks, fmt.Sprintf("domain: ERROR — %s", err.Error()))
			dr.Issues = append(dr.Issues, err.Error())
		} else {
			dr.Domain = domain
			dr.DomainSource = source
			hostname := fmt.Sprintf("%s-%s.%s", cfg.Subdomain, resolveInstanceID(cfg), domain)
			dr.PublicURL = fmt.Sprintf("https://%s", hostname)
			dr.Checks = append(dr.Checks, fmt.Sprintf("domain: %s (from %s)", domain, source))
			dr.Checks = append(dr.Checks, fmt.Sprintf("would add ingress: %s → localhost:%d", hostname, listenPort))
		}
	}

	return dr, nil
}

func (m *Manager) startLocked(tmp bool) error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.UpstreamProxy == "" {
		return newError(ErrNotConfigured, "upstream_proxy is not configured")
	}

	if cfg.UUID == "" {
		cfg.UUID = generateUUID()
		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	if cfg.InstanceID == "" {
		cfg.InstanceID = generateInstanceID()
		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	listenPort := resolvePort(cfg)
	if cfg.ListenPort != listenPort {
		cfg.ListenPort = listenPort
		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	m.hydrateFromConfig(cfg)

	mgr := subprocess.GetManager()
	if mgr.IsRunning(xrayProcID) {
		publicURL := m.effectivePublicURL(cfg)
		if m.isClientReady(cfg, publicURL, listenPort) {
			return newError(ErrAlreadyRunning, "ws-proxy is already running")
		}
		fmt.Printf("[ws-proxy] xray process running but tunnel degraded, recovering ingress...\n")
	}

	if !isXrayAlive(listenPort, cfg.WSPath) && !mgr.IsRunning(xrayProcID) {
		if err := ensureXrayBinary(); err != nil {
			return fmt.Errorf("failed to setup xray: %w", err)
		}

		if err := generateXrayConfig(cfg); err != nil {
			return fmt.Errorf("failed to generate xray config: %w", err)
		}

		xrayCmd := exec.Command(xrayBinaryPath(), "run", "-c", xrayConfigPath())
		xrayCmd.Stdout = os.Stdout
		xrayCmd.Stderr = os.Stderr

		healthCheck := func() bool {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", listenPort), 2*time.Second)
			if err != nil {
				return false
			}
			conn.Close()
			return true
		}

		proc, err := mgr.StartProcess(xrayProcID, "ws-proxy-xray", xrayCmd, healthCheck)
		if err != nil {
			return fmt.Errorf("failed to start xray: %w", err)
		}

		if !proc.WaitForRunning(30 * time.Second) {
			mgr.StopProcess(xrayProcID)
			return fmt.Errorf("xray failed to become healthy within 30s")
		}
	} else {
		fmt.Printf("[ws-proxy] xray already running on port %d, reusing\n", listenPort)
	}

	if tmp {
		if err := m.startQuickTunnel(cfg); err != nil {
			return err
		}
	} else {
		if err := m.startPermanentTunnel(cfg); err != nil {
			return err
		}
	}
	m.setAutoStart(true)
	return nil
}

func (m *Manager) startLockedStreaming(tmp bool, sw *sse.Writer) error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.UpstreamProxy == "" {
		return newError(ErrNotConfigured, "upstream_proxy is not configured")
	}

	sw.SendLog("Checking xray binary...")

	if cfg.UUID == "" {
		cfg.UUID = generateUUID()
		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	if cfg.InstanceID == "" {
		cfg.InstanceID = generateInstanceID()
		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	listenPort := resolvePort(cfg)
	if cfg.ListenPort != listenPort {
		cfg.ListenPort = listenPort
		if err := SaveConfig(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	m.hydrateFromConfig(cfg)

	mgr := subprocess.GetManager()
	if mgr.IsRunning(xrayProcID) {
		publicURL := m.effectivePublicURL(cfg)
		if m.isClientReady(cfg, publicURL, listenPort) {
			return newError(ErrAlreadyRunning, "ws-proxy is already running")
		}
		sw.SendLog("xray process running but tunnel degraded, recovering ingress...")
	}

	if !isXrayAlive(listenPort, cfg.WSPath) && !mgr.IsRunning(xrayProcID) {
		sw.SendLog(fmt.Sprintf("Ensuring xray binary at %s...", xrayBinaryPath()))
		if err := ensureXrayBinary(); err != nil {
			return fmt.Errorf("failed to setup xray: %w", err)
		}

		sw.SendLog("Generating xray config...")
		if err := generateXrayConfig(cfg); err != nil {
			return fmt.Errorf("failed to generate xray config: %w", err)
		}

		sw.SendLog(fmt.Sprintf("Starting xray on port %d...", listenPort))
		xrayCmd := exec.Command(xrayBinaryPath(), "run", "-c", xrayConfigPath())
		xrayCmd.Stdout = os.Stdout
		xrayCmd.Stderr = os.Stderr

		healthCheck := func() bool {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", listenPort), 2*time.Second)
			if err != nil {
				return false
			}
			conn.Close()
			return true
		}

		proc, err := mgr.StartProcess(xrayProcID, "ws-proxy-xray", xrayCmd, healthCheck)
		if err != nil {
			return fmt.Errorf("failed to start xray: %w", err)
		}

		sw.SendLog("Waiting for xray to become healthy...")
		if !proc.WaitForRunning(30 * time.Second) {
			mgr.StopProcess(xrayProcID)
			return newError(ErrStartupFailed, "xray failed to become healthy within 30s")
		}
		sw.SendLog("xray is healthy!")
	} else {
		sw.SendLog(fmt.Sprintf("xray already running on port %d, reusing", listenPort))
	}

	if tmp {
		if err := m.startQuickTunnelStreaming(cfg, sw); err != nil {
			return err
		}
	} else {
		if err := m.startPermanentTunnelStreaming(cfg, sw); err != nil {
			return err
		}
	}
	m.setAutoStart(true)
	return nil
}

func (m *Manager) startQuickTunnelStreaming(cfg *Config, sw *sse.Writer) error {
	cloudflaredCmd := exec.Command("cloudflared", "tunnel", "--url",
		fmt.Sprintf("http://localhost:%d", cfg.ListenPort))

	stderr, err := cloudflaredCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	cloudflaredCmd.Stdout = os.Stdout

	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)
	urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if match := urlRegex.FindString(line); match != "" {
				resultCh <- match
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		}
	}()

	_, err = subprocess.GetManager().StartProcess(cfQuickProcID, "ws-proxy-cf-quick", cloudflaredCmd,
		func() bool { return true })
	if err != nil {
		if !strings.Contains(err.Error(), "already running") {
			return fmt.Errorf("failed to start cloudflared: %w", err)
		}
	}

	var publicURL string
	select {
	case url := <-resultCh:
		publicURL = url
		sw.SendLog(fmt.Sprintf("Tunnel established: %s", publicURL))
	case err := <-errCh:
		subprocess.GetManager().StopProcess(cfQuickProcID)
		return fmt.Errorf("cloudflared error: %w", err)
	case <-time.After(60 * time.Second):
		subprocess.GetManager().StopProcess(cfQuickProcID)
		return fmt.Errorf("timeout waiting for cloudflared URL")
	}

	m.mu.Lock()
	m.publicURL = publicURL
	m.isTmp = true
	_ = m.persistRuntimeState(cfg)
	m.mu.Unlock()
	return nil
}

func (m *Manager) startPermanentTunnelStreaming(cfg *Config, sw *sse.Writer) error {
	domain, _, err := resolveDomainWithSource()
	if err != nil {
		return err
	}

	hostname := fmt.Sprintf("%s-%s.%s", cfg.Subdomain, cfg.InstanceID, domain)
	sw.SendLog(fmt.Sprintf("Adding ingress: %s → http://localhost:%d", hostname, cfg.ListenPort))
	if err := m.addPermanentTunnelMapping(cfg, hostname, cfg.ListenPort); err != nil {
		return err
	}
	sw.SendLog(fmt.Sprintf("Tunnel established: %s", m.publicURL))
	return nil
}

func (m *Manager) startQuickTunnel(cfg *Config) error {
	cloudflaredCmd := exec.Command("cloudflared", "tunnel", "--url",
		fmt.Sprintf("http://localhost:%d", cfg.ListenPort))

	stderr, err := cloudflaredCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	cloudflaredCmd.Stdout = os.Stdout

	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)
	urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if match := urlRegex.FindString(line); match != "" {
				resultCh <- match
				return
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		}
	}()

	_, err = subprocess.GetManager().StartProcess(cfQuickProcID, "ws-proxy-cf-quick", cloudflaredCmd,
		func() bool { return true })
	if err != nil {
		if !strings.Contains(err.Error(), "already running") {
			return fmt.Errorf("failed to start cloudflared: %w", err)
		}
	}

	var publicURL string
	select {
	case url := <-resultCh:
		publicURL = url
	case err := <-errCh:
		subprocess.GetManager().StopProcess(cfQuickProcID)
		return fmt.Errorf("cloudflared error: %w", err)
	case <-time.After(60 * time.Second):
		subprocess.GetManager().StopProcess(cfQuickProcID)
		return fmt.Errorf("timeout waiting for cloudflared URL")
	}

	m.mu.Lock()
	m.publicURL = publicURL
	m.isTmp = true
	_ = m.persistRuntimeState(cfg)
	m.mu.Unlock()
	return nil
}

func (m *Manager) startPermanentTunnel(cfg *Config) error {
	domain, err := resolveDomain()
	if err != nil {
		return err
	}

	hostname := fmt.Sprintf("%s-%s.%s", cfg.Subdomain, cfg.InstanceID, domain)
	return m.addPermanentTunnelMapping(cfg, hostname, cfg.ListenPort)
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	mgr := subprocess.GetManager()

	if mgr.IsRunning(cfQuickProcID) {
		mgr.StopProcess(cfQuickProcID)
	}

	if mgr.IsRunning(xrayProcID) {
		mgr.StopProcess(xrayProcID)
	}

	if !m.isTmp && m.publicURL != "" {
		tg := unified_tunnel.GetTunnelGroupManager().GetExtensionGroup()
		if tg != nil {
			_ = tg.RemoveMapping(mappingID)
		}
	}

	m.publicURL = ""
	m.isTmp = false
	if cfg, err := LoadConfig(); err == nil {
		m.clearPersistedRuntimeState(cfg)
	}
	m.setAutoStart(false)
	return nil
}

func (m *Manager) Status() *Status {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, _ := LoadConfig()
	m.hydrateFromConfig(cfg)
	port := resolvePort(cfg)
	publicURL := m.effectivePublicURL(cfg)
	running := m.isClientReady(cfg, publicURL, port)

	return &Status{
		Running:   running,
		PublicURL: publicURL,
		Port:      port,
		IsTmp:     m.isTmp,
	}
}

func (m *Manager) GetVMessLink() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, _ := LoadConfig()
	m.hydrateFromConfig(cfg)
	port := resolvePort(cfg)
	publicURL := m.effectivePublicURL(cfg)
	if !m.isClientReady(cfg, publicURL, port) {
		return ""
	}

	host := strings.TrimPrefix(publicURL, "https://")
	host = strings.TrimPrefix(host, "http://")

	vmess := map[string]interface{}{
		"v":    "2",
		"ps":   "ws-proxy",
		"add":  host,
		"port": "443",
		"id":   cfg.UUID,
		"aid":  "0",
		"net":  "ws",
		"type": "none",
		"host": host,
		"path": cfg.WSPath,
		"tls":  "tls",
	}

	data, _ := json.Marshal(vmess)
	return "vmess://" + base64.StdEncoding.EncodeToString(data)
}

func (m *Manager) GetVMessConfig() (*VMessConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	m.hydrateFromConfig(cfg)
	port := resolvePort(cfg)
	publicURL := m.effectivePublicURL(cfg)
	if !m.isClientReady(cfg, publicURL, port) {
		return nil, fmt.Errorf("ws-proxy is not running or not configured")
	}

	host := strings.TrimPrefix(publicURL, "https://")
	host = strings.TrimPrefix(host, "http://")

	return &VMessConfig{
		Host:    host,
		Port:    "443",
		UUID:    cfg.UUID,
		AlterID: "0",
		Network: "ws",
		Type:    "none",
		Path:    cfg.WSPath,
		TLS:     "tls",
	}, nil
}

func ensureXrayBinary() error {
	xrayPath := xrayBinaryPath()
	if _, err := os.Stat(xrayPath); err == nil {
		return nil
	}

	dir := xrayDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create xray dir: %w", err)
	}

	zipPath := filepath.Join(dir, "xray.zip")

	resp, err := http.Get(xrayDownloadURLAmd64)
	if err != nil {
		return fmt.Errorf("failed to download xray: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download xray: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return fmt.Errorf("failed to download xray: %w", err)
	}
	f.Close()

	data, err := os.ReadFile(zipPath)
	if err != nil {
		return fmt.Errorf("failed to read downloaded zip: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}

	for _, zf := range zipReader.File {
		if zf.Name == "xray" || zf.Name == "xray.exe" {
			rc, err := zf.Open()
			if err != nil {
				return fmt.Errorf("failed to open xray in zip: %w", err)
			}
			defer rc.Close()

			out, err := os.OpenFile(xrayPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("failed to create xray binary: %w", err)
			}
			defer out.Close()

			if _, err := io.Copy(out, rc); err != nil {
				return fmt.Errorf("failed to extract xray: %w", err)
			}

			os.Remove(zipPath)
			return nil
		}
	}

	return fmt.Errorf("xray binary not found in release zip")
}

func generateXrayConfig(cfg *Config) error {
	xrayCfg := xrayConfig{
		Inbounds: []xrayInbound{
			{
				Port:     cfg.ListenPort,
				Protocol: "vmess",
				Settings: xrayInboundSettings{
					Clients: []xrayClient{
						{ID: cfg.UUID, AlterID: 0},
					},
				},
				StreamSettings: xrayStreamSettings{
					Network: "ws",
					WSSettings: xrayWSSettings{
						Path: cfg.WSPath,
					},
				},
			},
		},
		Outbounds: []xrayOutbound{
			{
				Protocol: "http",
				Settings: xrayOutboundSettings{
					Servers: []xrayProxyServer{
						{
							Address: extractHost(cfg.UpstreamProxy),
							Port:    extractPort(cfg.UpstreamProxy),
						},
					},
				},
			},
		},
	}

	if err := os.MkdirAll(xrayDir(), 0755); err != nil {
		return fmt.Errorf("failed to create xray dir: %w", err)
	}

	data, err := json.MarshalIndent(xrayCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal xray config: %w", err)
	}

	return os.WriteFile(xrayConfigPath(), data, 0644)
}

func extractHost(proxyURL string) string {
	s := strings.TrimPrefix(proxyURL, "http://")
	s = strings.TrimPrefix(s, "https://")
	if idx := strings.LastIndex(s, ":"); idx != -1 {
		return s[:idx]
	}
	return s
}

func extractPort(proxyURL string) int {
	s := strings.TrimPrefix(proxyURL, "http://")
	s = strings.TrimPrefix(s, "https://")
	if idx := strings.LastIndex(s, ":"); idx != -1 {
		port := 3128
		fmt.Sscanf(s[idx+1:], "%d", &port)
		return port
	}
	return 3128
}

func resolveDomain() (string, error) {
	domain, _, err := resolveDomainWithSource()
	return domain, err
}

func resolveDomainWithSource() (string, string, error) {
	domains := cfutils.GetOwnedDomains()
	if len(domains) > 0 {
		return domains[0], "owned-domains config", nil
	}

	scfg := serverconfig.Get()
	if scfg != nil {
		for _, p := range scfg.PortForwarding.Providers {
			if p.Cloudflare != nil && p.Cloudflare.BaseDomain != "" {
				return p.Cloudflare.BaseDomain, "port-forwarding base_domain", nil
			}
		}
	}

	return "", "", fmt.Errorf(
		"no domain configured. set owned domains via /api/cloudflare/owned-domains " +
			"or configure a Cloudflare Tunnel base_domain, or use ?tmp=true for quick tunnel")
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func generateInstanceID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func resolveInstanceID(cfg *Config) string {
	if cfg.InstanceID != "" {
		return cfg.InstanceID
	}
	return generateInstanceID()
}

func resolvePort(cfg *Config) int {
	if cfg.ListenPort > 0 {
		if isXrayAlive(cfg.ListenPort, cfg.WSPath) {
			return cfg.ListenPort
		}
		if isPortFree(cfg.ListenPort) {
			return cfg.ListenPort
		}
	}
	return pickFreePort()
}

func isXrayAlive(port int, wsPath string) bool {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d%s", port, wsPath))
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == 400
}

func isPortFree(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 50*time.Millisecond)
	if err != nil {
		return true
	}
	conn.Close()
	return false
}

func pickFreePort() int {
	for i := 0; i < 100; i++ {
		port := portRangeLow + int(randInt(int64(portRangeHigh-portRangeLow)))
		if isPortFree(port) {
			return port
		}
	}
	return portRangeLow + int(randInt(int64(portRangeHigh-portRangeLow)))
}

func randInt(n int64) int64 {
	b := make([]byte, 8)
	rand.Read(b)
	v := int64(b[0])<<56 | int64(b[1])<<48 | int64(b[2])<<40 | int64(b[3])<<32 |
		int64(b[4])<<24 | int64(b[5])<<16 | int64(b[6])<<8 | int64(b[7])
	if v < 0 {
		v = -v
	}
	return v % n
}

func (m *Manager) setAutoStart(v bool) {
	cfg, err := LoadConfig()
	if err != nil {
		return
	}
	cfg.AutoStart = v
	SaveConfig(cfg)
}

var autoStartOnce sync.Once

func AutoStart() {
	autoStartOnce.Do(func() {
		go func() {
			cfg, err := LoadConfig()
			if err != nil {
				fmt.Printf("[ws-proxy] Auto-start: failed to load config: %v\n", err)
				return
			}
			if !cfg.AutoStart {
				fmt.Printf("[ws-proxy] Auto-start: disabled by config\n")
				return
			}
			if cfg.UpstreamProxy == "" {
				fmt.Printf("[ws-proxy] Auto-start: upstream_proxy not configured\n")
				return
			}

			tg := unified_tunnel.GetTunnelGroupManager().GetExtensionGroup()
			if tg.GetConfig() == nil {
				fmt.Printf("[ws-proxy] Auto-start: waiting for extension tunnel config...\n")
				select {
				case <-unified_tunnel.WaitExtensionConfig():
					fmt.Printf("[ws-proxy] Auto-start: extension tunnel ready, starting...\n")
				case <-time.After(30 * time.Second):
					fmt.Printf("[ws-proxy] Auto-start: timed out waiting for extension tunnel config\n")
					return
				}
			}

			m := GetManager()
			if err := m.Recover(); err != nil {
				fmt.Printf("[ws-proxy] Auto-recover failed: %v\n", err)
			}
			status := m.Status()
			if status.Running {
				fmt.Printf("[ws-proxy] Auto-recovered: %s\n", status.PublicURL)
				return
			}
			if err := m.Start(false); err != nil {
				fmt.Printf("[ws-proxy] Auto-start failed: %v\n", err)
				return
			}
			status = m.Status()
			fmt.Printf("[ws-proxy] Auto-started: %s\n", status.PublicURL)
		}()
	})
}
