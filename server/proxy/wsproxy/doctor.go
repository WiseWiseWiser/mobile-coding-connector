package wsproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/server/cloudflare/unified_tunnel"
	"github.com/xhd2015/ai-critic/server/streaming/progress"
	"github.com/xhd2015/ai-critic/server/subprocess"
)

const defaultDoctorTryURL = "https://www.google.com"

type DoctorCheckStatus string

const (
	DoctorOK    DoctorCheckStatus = "ok"
	DoctorWarn  DoctorCheckStatus = "warn"
	DoctorFail  DoctorCheckStatus = "fail"
	DoctorSkip  DoctorCheckStatus = "skip"
)

type DoctorCheck struct {
	ID     string            `json:"id"`
	Layer  string            `json:"layer"`
	Name   string            `json:"name"`
	Status DoctorCheckStatus `json:"status"`
	Detail string            `json:"detail,omitempty"`
	Hint   string            `json:"hint,omitempty"`
}

type DoctorReport struct {
	Healthy bool         `json:"healthy"`
	TryURL  string       `json:"try_url"`
	Status  *Status      `json:"status,omitempty"`
	VMess   *VMessConfig `json:"vmess,omitempty"`
	Checks  []DoctorCheck `json:"checks"`
}

// DoctorCheckEmitter is invoked immediately when each server doctor check completes.
type DoctorCheckEmitter func(DoctorCheck)

func (m *Manager) Doctor(tryURL string) *DoctorReport {
	if strings.TrimSpace(tryURL) == "" {
		tryURL = defaultDoctorTryURL
	}
	if _, err := url.Parse(tryURL); err != nil {
		return &DoctorReport{
			Healthy: false,
			TryURL:  tryURL,
			Checks: []DoctorCheck{{
				ID: "try_url", Layer: "server", Name: "try URL",
				Status: DoctorFail, Detail: err.Error(),
			}},
		}
	}

	_ = m.Recover()

	report := &DoctorReport{TryURL: tryURL}
	report.Status = m.Status()
	report.Checks = append(report.Checks, m.serverDoctorChecks(tryURL, nil)...)

	if report.Status != nil && report.Status.Running {
		if vmess, err := m.GetVMessConfig(); err == nil {
			report.VMess = vmess
		}
	}

	report.Healthy = true
	for _, c := range report.Checks {
		if c.Status == DoctorFail {
			report.Healthy = false
			break
		}
	}
	return report
}

func (m *Manager) DoctorStream(w http.ResponseWriter, tryURL string) error {
	pw := progress.NewWriter(w)
	if pw == nil {
		return fmt.Errorf("streaming not supported")
	}

	if strings.TrimSpace(tryURL) == "" {
		tryURL = defaultDoctorTryURL
	}
	if _, err := url.Parse(tryURL); err != nil {
		chk := DoctorCheck{
			ID: "try_url", Layer: "server", Name: "try URL",
			Status: DoctorFail, Detail: err.Error(),
		}
		_ = pw.EmitProgress(doctorCheckToItem(chk))
		_ = pw.EmitDone(map[string]any{
			"healthy":        false,
			"try_url":        tryURL,
			"checks_total":   1,
			"checks_failed":  1,
		})
		return nil
	}

	_ = m.Recover()
	status := m.Status()

	_ = pw.EmitMeta(map[string]any{
		"message": "WS Proxy Doctor",
		"try_url": tryURL,
	})
	if status != nil {
		_ = pw.EmitMeta(map[string]any{
			"server_status": map[string]any{
				"running":    status.Running,
				"public_url": status.PublicURL,
				"port":       status.Port,
				"is_tmp":     status.IsTmp,
			},
		})
	}
	_ = pw.EmitSection("Server checks")

	checks := m.serverDoctorChecks(tryURL, func(chk DoctorCheck) {
		_ = pw.EmitProgress(doctorCheckToItem(chk))
	})

	var vmess *VMessConfig
	if status != nil && status.Running {
		if v, err := m.GetVMessConfig(); err == nil {
			vmess = v
		}
	}

	healthy, failed := doctorAggregateHealth(checks)
	done := map[string]any{
		"healthy":        healthy,
		"try_url":        tryURL,
		"checks_total":   len(checks),
		"checks_failed":  failed,
	}
	if status != nil {
		if raw, err := json.Marshal(status); err == nil {
			var statusMap map[string]any
			if json.Unmarshal(raw, &statusMap) == nil {
				done["status"] = statusMap
			}
		}
	}
	if vmess != nil {
		if raw, err := json.Marshal(vmess); err == nil {
			var vmessMap map[string]any
			if json.Unmarshal(raw, &vmessMap) == nil {
				done["vmess"] = vmessMap
			}
		}
	}
	return pw.EmitDone(done)
}

func doctorCheckToItem(chk DoctorCheck) progress.Item {
	return progress.Item{
		ID:     chk.ID,
		Layer:  chk.Layer,
		Name:   chk.Name,
		Status: string(chk.Status),
		Detail: chk.Detail,
		Hint:   chk.Hint,
	}
}

func doctorAggregateHealth(checks []DoctorCheck) (healthy bool, failed int) {
	healthy = true
	for _, c := range checks {
		if c.Status == DoctorFail {
			healthy = false
			failed++
		}
	}
	return healthy, failed
}

func (m *Manager) serverDoctorChecks(tryURL string, emit DoctorCheckEmitter) []DoctorCheck {
	var checks []DoctorCheck

	add := func(id, name string, status DoctorCheckStatus, detail, hint string) {
		chk := DoctorCheck{
			ID: id, Layer: "server", Name: name,
			Status: status, Detail: detail, Hint: hint,
		}
		checks = append(checks, chk)
		if emit != nil {
			emit(chk)
		}
	}
	appendCheck := func(chk DoctorCheck) {
		checks = append(checks, chk)
		if emit != nil {
			emit(chk)
		}
	}

	cfg, err := LoadConfig()
	if err != nil {
		add("config_load", "configuration load", DoctorFail, err.Error(),
			"check ws-proxy.json in the server data directory")
		return checks
	}
	add("config_load", "configuration load", DoctorOK, configPath(), "")

	if cfg.UpstreamProxy == "" {
		add("upstream_proxy", "upstream proxy configured", DoctorFail, "not set",
			"run: remote-agent ws-proxy config set --upstream-proxy URL")
	} else {
		add("upstream_proxy", "upstream proxy configured", DoctorOK, cfg.UpstreamProxy, "")
	}

	if cfg.UUID == "" {
		add("uuid", "VMess UUID configured", DoctorWarn, "not set (generated on start)", "")
	} else {
		add("uuid", "VMess UUID configured", DoctorOK, cfg.UUID, "")
	}

	xrayPath := xrayBinaryPath()
	if _, err := os.Stat(xrayPath); err != nil {
		add("xray_binary", "xray binary present", DoctorWarn,
			fmt.Sprintf("missing at %s (downloaded on start)", xrayPath), "")
	} else {
		add("xray_binary", "xray binary present", DoctorOK, xrayPath, "")
	}

	m.hydrateFromConfig(cfg)
	port := resolvePort(cfg)
	publicURL := m.effectivePublicURL(cfg)

	mgr := subprocess.GetManager()
	if mgr.IsRunning(xrayProcID) {
		add("xray_process", "xray process tracked", DoctorOK, xrayProcID, "")
	} else {
		add("xray_process", "xray process tracked", DoctorWarn,
			"not managed by ws-proxy subprocess manager", "")
	}

	if m.isLocalXrayAlive(cfg, port) {
		add("local_xray_health", "local xray WebSocket health", DoctorOK,
			fmt.Sprintf("GET http://127.0.0.1:%d%s → 400", port, cfg.WSPath), "")
	} else {
		add("local_xray_health", "local xray WebSocket health", DoctorFail,
			fmt.Sprintf("no healthy xray on port %d path %s", port, cfg.WSPath),
			"run: remote-agent ws-proxy start")
	}

	add("listen_port", "xray listen port", DoctorOK, fmt.Sprintf("%d", port), "")

	if publicURL == "" {
		add("public_url", "public URL known", DoctorFail, "empty",
			"run: remote-agent ws-proxy start")
	} else {
		add("public_url", "public URL known", DoctorOK, publicURL, "")
	}

	if m.isTmp {
		if mgr.IsRunning(cfQuickProcID) {
			add("quick_tunnel", "Cloudflare quick tunnel process", DoctorOK, cfQuickProcID, "")
		} else {
			add("quick_tunnel", "Cloudflare quick tunnel process", DoctorFail,
				"cloudflared quick tunnel not running",
				"run: remote-agent ws-proxy start --tmp")
		}
		add("extension_tunnel", "extension Cloudflare tunnel config", DoctorSkip,
			"not required for quick tunnel", "")
		add("tunnel_ingress", "tunnel ingress mapping", DoctorSkip,
			"not required for quick tunnel", "")
	} else {
		add("quick_tunnel", "Cloudflare quick tunnel process", DoctorSkip,
			"permanent tunnel mode", "")

		tg := unified_tunnel.GetTunnelGroupManager().GetExtensionGroup()
		tgCfg := tg.GetConfig()
		if tgCfg == nil {
			add("extension_tunnel", "extension Cloudflare tunnel config", DoctorFail,
				"extension tunnel is not configured",
				"configure Cloudflare extension tunnel on the server")
		} else {
			ref := tgCfg.TunnelName
			if ref == "" {
				ref = tgCfg.TunnelID
			}
			add("extension_tunnel", "extension Cloudflare tunnel config", DoctorOK, ref, "")
			appendCheck(checkStaleTunnelConnectors(tg))
		}

		if publicURL == "" {
			add("tunnel_ingress", "tunnel ingress mapping", DoctorFail,
				"public URL unknown", "")
		} else if hasTunnelMapping(HostFromPublicURL(publicURL), port) {
			hostname := HostFromPublicURL(publicURL)
			add("tunnel_ingress", "tunnel ingress mapping", DoctorOK,
				fmt.Sprintf("%s → http://localhost:%d", hostname, port), "")
		} else {
			add("tunnel_ingress", "tunnel ingress mapping", DoctorFail,
				fmt.Sprintf("missing ws-proxy mapping for %s", HostFromPublicURL(publicURL)),
				"run: remote-agent ws-proxy status (auto-recover) or ws-proxy stop && start")
		}
	}

	if publicURL != "" {
		appendCheck(checkPublicWSEndpoint(publicURL, cfg.WSPath))
	} else {
		add("public_ws_endpoint", "public WebSocket endpoint", DoctorSkip,
			"no public URL", "")
	}

	if cfg.UpstreamProxy != "" {
		appendCheck(checkUpstreamTCP(cfg.UpstreamProxy))
		appendCheck(checkUpstreamFetch(cfg.UpstreamProxy, tryURL))
	} else {
		add("upstream_tcp", "upstream proxy TCP reachability", DoctorSkip,
			"upstream proxy not configured", "")
		add("upstream_fetch", "upstream proxy fetch test", DoctorSkip,
			"upstream proxy not configured", "")
	}

	clientReady := m.isClientReady(cfg, publicURL, port)
	if clientReady {
		add("client_ready", "server client-ready state", DoctorOK,
			"xray + tunnel + public URL + UUID", "")
	} else {
		add("client_ready", "server client-ready state", DoctorFail,
			"proxy not ready for VMess clients",
			"fix failing checks above, then retry doctor")
	}

	return checks
}

func doctorDirectHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			// Bypass HTTP_PROXY/HTTPS_PROXY so we test the public path like an external client.
			Proxy: func(*http.Request) (*url.URL, error) { return nil, nil },
		},
	}
}

func serverProxyEnvSummary() string {
	var parts []string
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", key, v))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

func checkPublicWSEndpoint(publicURL, wsPath string) DoctorCheck {
	hostname := HostFromPublicURL(publicURL)
	target := fmt.Sprintf("https://%s%s", hostname, wsPath)

	if _testStubNetworkChecks {
		return DoctorCheck{
			ID: "public_ws_endpoint", Layer: "server",
			Name: "public WebSocket endpoint (direct egress)", Status: DoctorOK,
			Detail: fmt.Sprintf("%s → HTTP 400 (stubbed)", target),
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return DoctorCheck{
			ID: "public_ws_endpoint", Layer: "server",
			Name: "public WebSocket endpoint (direct egress)", Status: DoctorFail,
			Detail: err.Error(),
		}
	}

	client := doctorDirectHTTPClient(15 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return DoctorCheck{
			ID: "public_ws_endpoint", Layer: "server",
			Name: "public WebSocket endpoint (direct egress)", Status: DoctorFail,
			Detail: fmt.Sprintf("%s: %v", target, err),
			Hint:   "Cloudflare tunnel may not be routing; check tunnel ingress mapping",
		}
	}
	defer resp.Body.Close()

	detail := fmt.Sprintf("%s → HTTP %d (direct egress, ignores HTTP_PROXY)", target, resp.StatusCode)
	if proxyEnv := serverProxyEnvSummary(); proxyEnv != "" {
		detail += "; server env has " + proxyEnv
	}

	if resp.StatusCode == http.StatusBadRequest {
		return DoctorCheck{
			ID: "public_ws_endpoint", Layer: "server",
			Name: "public WebSocket endpoint (direct egress)", Status: DoctorOK, Detail: detail,
		}
	}

	hint := "expected HTTP 400 from xray; HTTP 404 usually means missing Cloudflare ingress"
	if proxyEnv := serverProxyEnvSummary(); proxyEnv != "" {
		hint = "server has " + proxyEnv + " set; an older doctor run used DefaultClient and could falsely report HTTP 404 via the corporate proxy while external clients see HTTP 400"
	}
	return DoctorCheck{
		ID: "public_ws_endpoint", Layer: "server",
		Name: "public WebSocket endpoint (direct egress)", Status: DoctorFail,
		Detail: detail, Hint: hint,
	}
}

func checkUpstreamTCP(proxyURL string) DoctorCheck {
	if _testStubNetworkChecks {
		return DoctorCheck{
			ID: "upstream_tcp", Layer: "server",
			Name: "upstream proxy TCP reachability", Status: DoctorOK,
			Detail: extractHost(proxyURL) + " (stubbed)",
		}
	}

	host := extractHost(proxyURL)
	port := extractPort(proxyURL)
	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return DoctorCheck{
			ID: "upstream_tcp", Layer: "server",
			Name: "upstream proxy TCP reachability", Status: DoctorFail,
			Detail: fmt.Sprintf("%s: %v", addr, err),
			Hint:   "verify upstream proxy is running and reachable from the server",
		}
	}
	conn.Close()
	return DoctorCheck{
		ID: "upstream_tcp", Layer: "server",
		Name: "upstream proxy TCP reachability", Status: DoctorOK,
		Detail: addr,
	}
}

func checkUpstreamFetch(proxyURL, tryURL string) DoctorCheck {
	if _testUpstreamFetchDelay > 0 {
		time.Sleep(_testUpstreamFetchDelay)
	}

	if _testStubNetworkChecks {
		return DoctorCheck{
			ID: "upstream_fetch", Layer: "server",
			Name: "upstream proxy fetch test", Status: DoctorOK,
			Detail: fmt.Sprintf("GET %s via %s → HTTP 200 (stubbed)", tryURL, proxyURL),
		}
	}

	proxy, err := url.Parse(proxyURL)
	if err != nil {
		return DoctorCheck{
			ID: "upstream_fetch", Layer: "server",
			Name: "upstream proxy fetch test", Status: DoctorFail,
			Detail: err.Error(),
		}
	}

	transport := &http.Transport{Proxy: http.ProxyURL(proxy)}
	client := &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(tryURL)
	if err != nil {
		return DoctorCheck{
			ID: "upstream_fetch", Layer: "server",
			Name: "upstream proxy fetch test", Status: DoctorFail,
			Detail: fmt.Sprintf("GET %s via %s: %v", tryURL, proxyURL, err),
			Hint:   "xray forwards traffic through upstream_proxy; fix upstream connectivity first",
		}
	}
	defer resp.Body.Close()

	detail := fmt.Sprintf("GET %s via %s → HTTP %d", tryURL, proxyURL, resp.StatusCode)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return DoctorCheck{
			ID: "upstream_fetch", Layer: "server",
			Name: "upstream proxy fetch test", Status: DoctorOK, Detail: detail,
		}
	}
	return DoctorCheck{
		ID: "upstream_fetch", Layer: "server",
		Name: "upstream proxy fetch test", Status: DoctorFail,
		Detail: detail,
		Hint:   "upstream proxy responded but returned an error status",
	}
}

func checkStaleTunnelConnectors(tg *unified_tunnel.TunnelGroup) DoctorCheck {
	check := DoctorCheck{
		ID: "stale_tunnel_connectors", Layer: "server",
		Name: "stale cloudflared tunnel connectors",
	}
	if tg == nil {
		check.Status = DoctorSkip
		check.Detail = "extension tunnel group unavailable"
		return check
	}

	killed, err := tg.TunnelMgr().ReconcileStaleConnectors()
	if err != nil {
		check.Status = DoctorFail
		check.Detail = err.Error()
		check.Hint = "remove orphan cloudflared processes for the extension tunnel"
		return check
	}
	if len(killed) == 0 {
		check.Status = DoctorOK
		check.Detail = "no stale connectors"
		return check
	}

	check.Status = DoctorOK
	check.Detail = fmt.Sprintf("removed stale connector PIDs: %s", strings.TrimSpace(
		strings.Join(intSliceToString(killed), ", ")))
	return check
}