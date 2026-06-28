package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/cmd/remote-agent/streamcmd"
	"github.com/xhd2015/ai-critic/cmd/remote-agent/wsproxy_singbox"
	"github.com/xhd2015/less-gen/flags"
)

const (
	defaultDoctorURL     = "https://www.google.com"
	doctorRequestTimeout = 25 * time.Second

	xrayReleasesURL  = "https://github.com/XTLS/Xray-core/releases"
	v2rayUReleasesURL = "https://github.com/yanue/V2rayU/releases"
)

const xrayInstallHint = `Install Xray CLI (for doctor end-to-end test):
  macOS (Homebrew):  brew install xray
  macOS (manual):    download from ` + xrayReleasesURL + ` and put xray on PATH
  Linux (manual):    download from ` + xrayReleasesURL

const v2rayUInstallHint = `Or use V2RayU GUI on macOS (typical for daily use):
  Homebrew:  brew install --cask v2rayu
  Releases:  ` + v2rayUReleasesURL + `
  Then:      remote-agent ws-proxy vmess-link  → paste link in V2RayU → enable system proxy`

type doctorCheck struct {
	ID     string `json:"id"`
	Layer  string `json:"layer"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	Hint   string `json:"hint,omitempty"`
}

type doctorVMess struct {
	Host    string `json:"host"`
	Port    string `json:"port"`
	UUID    string `json:"uuid"`
	AlterID string `json:"alter_id"`
	Network string `json:"network"`
	Type    string `json:"type"`
	Path    string `json:"path"`
	TLS     string `json:"tls"`
}

type doctorReport struct {
	Healthy bool          `json:"healthy"`
	TryURL  string        `json:"try_url"`
	Status  map[string]any  `json:"status,omitempty"`
	VMess   *doctorVMess    `json:"vmess,omitempty"`
	Checks  []doctorCheck   `json:"checks"`
}

const wsproxyDoctorHelp = `Usage: remote-agent ws-proxy doctor [--try-url URL]

  Diagnose ws-proxy health on the server and from this machine (client).
  By default fetches https://www.google.com through the VMess proxy path.

Options:
  --try-url URL   URL to fetch via upstream (server) and VMess proxy (client)
  -h, --help      show this help
`

func wsproxyDoctor(getClient func() (*client.Client, error), args []string) error {
	var tryURL string
	_, err := flags.
		String("--try-url", &tryURL).
		Help("-h,--help", wsproxyDoctorHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if strings.TrimSpace(tryURL) == "" {
		tryURL = defaultDoctorURL
	}
	if _, err := url.Parse(tryURL); err != nil {
		return fmt.Errorf("invalid --try-url: %w", err)
	}

	return streamcmd.Run(getClient, streamcmd.Spec{
		Method: http.MethodGet,
		Path:   "/api/ws-proxy/doctor/stream",
		Query:  url.Values{"try_url": {tryURL}},
		Print:  streamcmd.Sections | streamcmd.Meta | streamcmd.ProgressChecks,
		After:  func(done map[string]any) error {
			return finishDoctorAfter(done, tryURL)
		},
	})
}

func finishDoctorAfter(done map[string]any, tryURL string) error {
	serverReport := doctorReportFromDone(done, tryURL)
	clientChecks := runClientDoctorChecks(serverReport, tryURL)

	fmt.Println()
	fmt.Println("Client checks:")
	if len(clientChecks) == 0 {
		fmt.Println("  (none)")
	} else {
		for _, chk := range clientChecks {
			_ = streamcmd.PrintProgress(doctorCheckToStreamEvent(chk))
		}
	}

	healthy := serverReport.Healthy
	for _, chk := range clientChecks {
		if chk.Status == "fail" {
			healthy = false
			break
		}
	}

	fmt.Println()
	if healthy {
		fmt.Println("Result: healthy")
		return nil
	}
	fmt.Println("Result: unhealthy (see [fail] items above)")
	return fmt.Errorf("ws-proxy doctor found failing checks")
}

func doctorReportFromDone(done map[string]any, tryURL string) *doctorReport {
	report := &doctorReport{TryURL: tryURL, Healthy: true}
	if done == nil {
		return report
	}
	if h, ok := done["healthy"].(bool); ok {
		report.Healthy = h
	}
	if u, ok := done["try_url"].(string); ok && u != "" {
		report.TryURL = u
	}
	if status, ok := done["status"].(map[string]any); ok {
		report.Status = status
	}
	if vmess, ok := done["vmess"].(map[string]any); ok {
		report.VMess = vmessFromMap(vmess)
	}
	return report
}

func vmessFromMap(m map[string]any) *doctorVMess {
	if m == nil {
		return nil
	}
	return &doctorVMess{
		Host:    stringFromAny(m["host"]),
		Port:    stringFromAny(m["port"]),
		UUID:    stringFromAny(m["uuid"]),
		AlterID: stringFromAny(m["alter_id"]),
		Network: stringFromAny(m["network"]),
		Type:    stringFromAny(m["type"]),
		Path:    stringFromAny(m["path"]),
		TLS:     stringFromAny(m["tls"]),
	}
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func doctorCheckToStreamEvent(chk doctorCheck) client.StreamEvent {
	return client.StreamEvent{
		Type:   "progress",
		ID:     chk.ID,
		Layer:  chk.Layer,
		Name:   chk.Name,
		Status: chk.Status,
		Detail: chk.Detail,
		Hint:   chk.Hint,
	}
}

func fetchServerDoctor(c *client.Client, tryURL string) (*doctorReport, error) {
	path := "/api/ws-proxy/doctor?try_url=" + url.QueryEscape(tryURL)
	req, err := c.NewRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call ws-proxy doctor API: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return nil, fmt.Errorf("%s", mapErrorToCLI(*apiErr))
	}

	var report doctorReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse doctor response: %w", err)
	}
	return &report, nil
}

func runClientDoctorChecks(server *doctorReport, tryURL string) []doctorCheck {
	var checks []doctorCheck
	add := func(id, name, status, detail, hint string) {
		checks = append(checks, doctorCheck{
			ID: id, Layer: "client", Name: name,
			Status: status, Detail: detail, Hint: hint,
		})
	}

	if server.VMess == nil || server.VMess.Host == "" {
		add("vmess_config", "VMess config from server", "skip",
			"server proxy not client-ready", "fix server checks first")
		add("dns_resolve", "public hostname DNS", "skip", "no public hostname", "")
		add("tls_connect", "TLS to public endpoint", "skip", "no public hostname", "")
		add("public_ws_endpoint", "public WebSocket endpoint", "skip", "no public hostname", "")
		add("vmess_proxy_fetch", "VMess proxy fetch test", "skip", "no VMess config", "")
		return checks
	}

	host := server.VMess.Host
	wsPath := server.VMess.Path
	if wsPath == "" {
		wsPath = "/ws"
	}

	if ips, err := net.LookupHost(host); err != nil {
		add("dns_resolve", "public hostname DNS", "fail",
			fmt.Sprintf("%s: %v", host, err),
			"verify DNS for the ws-proxy hostname")
	} else {
		add("dns_resolve", "public hostname DNS", "ok",
			fmt.Sprintf("%s → %s", host, strings.Join(ips, ", ")), "")
	}

	tlsAddr := net.JoinHostPort(host, "443")
	tlsConn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", tlsAddr, &tls.Config{
		ServerName: host,
	})
	if err != nil {
		add("tls_connect", "TLS to public endpoint", "fail",
			fmt.Sprintf("%s: %v", tlsAddr, err), "")
	} else {
		_ = tlsConn.Close()
		add("tls_connect", "TLS to public endpoint", "ok", tlsAddr, "")
	}

	checks = append(checks, clientPublicWSEndpointCheck(host, wsPath))

	fetchCheck, err := clientVMessProxyFetch(server.VMess, tryURL)
	if err != nil {
		add("xray_client", "local xray client for test", "warn",
			err.Error(), xrayInstallHint)
		add("vmess_proxy_fetch", "VMess proxy fetch test", "skip",
			"local xray client unavailable", v2rayUInstallHint)
	} else {
		checks = append(checks, fetchCheck...)
	}

	return checks
}

func clientPublicWSEndpointCheck(host, wsPath string) doctorCheck {
	target := fmt.Sprintf("https://%s%s", host, wsPath)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return doctorCheck{
			ID: "public_ws_endpoint", Layer: "client",
			Name: "public WebSocket endpoint (this machine)", Status: "fail", Detail: err.Error(),
		}
	}
	client := doctorDirectHTTPClient(15 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return doctorCheck{
			ID: "public_ws_endpoint", Layer: "client",
			Name: "public WebSocket endpoint (this machine)", Status: "fail",
			Detail: fmt.Sprintf("%s: %v", target, err),
			Hint:   "this Mac cannot reach the public ws-proxy endpoint",
		}
	}
	defer resp.Body.Close()

	detail := fmt.Sprintf("%s → HTTP %d (direct egress, ignores HTTP_PROXY)", target, resp.StatusCode)
	if resp.StatusCode == http.StatusBadRequest {
		return doctorCheck{
			ID: "public_ws_endpoint", Layer: "client",
			Name: "public WebSocket endpoint (this machine)", Status: "ok", Detail: detail,
		}
	}
	return doctorCheck{
		ID: "public_ws_endpoint", Layer: "client",
		Name: "public WebSocket endpoint (this machine)", Status: "fail", Detail: detail,
		Hint:   "expected HTTP 400 from xray; HTTP 404 means Cloudflare ingress is missing",
	}
}

func clientVMessProxyFetch(vmess *doctorVMess, tryURL string) ([]doctorCheck, error) {
	xrayPath, err := findDoctorXrayBinary()
	if err != nil {
		return nil, err
	}

	inboundPort, err := pickDoctorLocalPort()
	if err != nil {
		return nil, err
	}

	tmpDir, err := os.MkdirTemp("", "remote-agent-wsproxy-doctor-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	cfgPath := filepath.Join(tmpDir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(wsproxy_singbox.BuildXrayVMessClientConfig(doctorVMessToParams(vmess), inboundPort)), 0600); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, xrayPath, "run", "-c", cfgPath)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start xray: %w", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", inboundPort)
	if err := waitForDoctorProxy(proxyURL, 15*time.Second); err != nil {
		return []doctorCheck{{
			ID: "xray_client", Layer: "client",
			Name: "local xray client for test", Status: "fail",
			Detail: err.Error(),
		}}, nil
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(mustParseURL(proxyURL)),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   doctorRequestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(tryURL)
	if err != nil {
		return []doctorCheck{
			{
				ID: "xray_client", Layer: "client",
				Name: "local xray client for test", Status: "ok",
				Detail: fmt.Sprintf("%s listening on %s", xrayPath, proxyURL),
			},
			{
				ID: "vmess_proxy_fetch", Layer: "client",
				Name: "VMess proxy fetch test", Status: "fail",
				Detail: fmt.Sprintf("GET %s via VMess: %v", tryURL, err),
				Hint:   "matches V2Ray/V2RayU failure; fix server tunnel or VMess settings",
			},
		}, nil
	}
	defer resp.Body.Close()

	detail := fmt.Sprintf("GET %s via VMess → HTTP %d", tryURL, resp.StatusCode)
	status := "ok"
	hint := ""
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		status = "fail"
		hint = "proxy connected but target URL returned an error status"
	}

	return []doctorCheck{
		{
			ID: "xray_client", Layer: "client",
			Name: "local xray client for test", Status: "ok",
			Detail: fmt.Sprintf("%s listening on %s", xrayPath, proxyURL),
		},
		{
			ID: "vmess_proxy_fetch", Layer: "client",
			Name: "VMess proxy fetch test", Status: status,
			Detail: detail, Hint: hint,
		},
	}, nil
}

func doctorVMessToParams(vmess *doctorVMess) *wsproxy_singbox.VMessParams {
	if vmess == nil {
		return nil
	}
	return &wsproxy_singbox.VMessParams{
		Host:    vmess.Host,
		Port:    vmess.Port,
		UUID:    vmess.UUID,
		AlterID: vmess.AlterID,
		Network: vmess.Network,
		Type:    vmess.Type,
		Path:    vmess.Path,
		TLS:     vmess.TLS,
	}
}

func findDoctorXrayBinary() (string, error) {
	if p, err := exec.LookPath("xray"); err == nil {
		return p, nil
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("xray not found in PATH (%w)", err)
	}
	cached := filepath.Join(cacheDir, "remote-agent", "xray", xrayDoctorBinaryName())
	if _, err := os.Stat(cached); err == nil {
		return cached, nil
	}
	return "", fmt.Errorf("xray not found in PATH")
}

func xrayDoctorBinaryName() string {
	if runtime.GOOS == "windows" {
		return "xray.exe"
	}
	return "xray"
}

func pickDoctorLocalPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port, nil
}

func waitForDoctorProxy(proxyURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	proxy := mustParseURL(proxyURL)
	transport := &http.Transport{Proxy: http.ProxyURL(proxy)}
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/", nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for local xray proxy at %s", proxyURL)
}

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func printDoctorReport(tryURL string, status map[string]any, checks []doctorCheck, healthy bool) {
	fmt.Println("WS Proxy Doctor")
	fmt.Printf("Try URL: %s\n", tryURL)
	if status != nil {
		fmt.Printf("Server status: running=%v public_url=%v port=%v temporary=%v\n",
			status["running"], status["public_url"], status["port"], status["is_tmp"])
	}
	fmt.Println()

	var serverChecks, clientChecks []doctorCheck
	for _, c := range checks {
		if c.Layer == "client" {
			clientChecks = append(clientChecks, c)
		} else {
			serverChecks = append(serverChecks, c)
		}
	}

	printDoctorSection("Server checks", serverChecks)
	printDoctorSection("Client checks", clientChecks)

	fmt.Println()
	if healthy {
		fmt.Println("Result: healthy")
	} else {
		fmt.Println("Result: unhealthy (see [fail] items above)")
	}
}

func printDoctorSection(title string, checks []doctorCheck) {
	fmt.Println(title + ":")
	if len(checks) == 0 {
		fmt.Println("  (none)")
		return
	}
	for _, c := range checks {
		tag := doctorStatusTag(c.Status)
		line := fmt.Sprintf("  %s  %s", tag, c.Name)
		if c.Detail != "" {
			line += ": " + c.Detail
		}
		fmt.Println(line)
		if c.Hint != "" {
			fmt.Printf("         hint:\n%s\n", indentDoctorHint(c.Hint))
		}
	}
}

func indentDoctorHint(hint string) string {
	lines := strings.Split(strings.TrimSpace(hint), "\n")
	for i, line := range lines {
		lines[i] = "           " + line
	}
	return strings.Join(lines, "\n")
}

func doctorStatusTag(status string) string {
	switch status {
	case "ok":
		return "[ok]  "
	case "warn":
		return "[warn]"
	case "skip":
		return "[skip]"
	default:
		return "[fail]"
	}
}

func doctorDirectHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: func(*http.Request) (*url.URL, error) { return nil, nil },
		},
	}
}