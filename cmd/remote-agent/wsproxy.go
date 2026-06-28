package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/skip2/go-qrcode"
	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const wsproxyHelp = `Usage: remote-agent ws-proxy <subcommand> [args...]

Manage the WebSocket-based proxy for accessing internal sites from mobile.
Uses Xray (VMess + WebSocket) routed through Cloudflare Tunnel.

Subcommands:
  start [--tmp] [--dry-run] [--upstream-proxy URL]
      Start the ws-proxy. With --tmp, uses a temporary Cloudflare Quick
      Tunnel (trycloudflare.com) — no domain required, everything is
      discarded on exit. Without --tmp, requires a configured domain
      and uses the permanent Cloudflare Tunnel.
      With --dry-run, validates configuration and shows what would happen
      without making any changes.

  stop
      Stop the ws-proxy.

  status
      Show ws-proxy status (running, public URL, port).

  config
      Show current configuration.

  config set --upstream-proxy URL [--port PORT] [--path PATH] [--subdomain SUB]
      Update ws-proxy configuration.

  vmess-link [--export FILE]
      Get the vmess:// link, manual config, and QR code for Shadowrocket import.

  doctor [--try-url URL]
      Diagnose server and client proxy health. Default fetch test uses
      https://www.google.com through the VMess path.

  sing-box client-config [--output FILE]
      Generate sing-box TUN config from VMess params.

  sing-box run-tun [--yes] [--no-install] [--config FILE] [--detach]
      Start sing-box TUN tunnel for ws-proxy.

  vpn [--yes] [--no-install] [--config FILE] [--detach]
      Alias for sing-box run-tun — start a system-wide TUN mini-VPN.

Examples:
  remote-agent ws-proxy start --tmp
  remote-agent ws-proxy start --tmp --upstream-proxy http://squid.internal:3128
  remote-agent ws-proxy start
  remote-agent ws-proxy start --dry-run
  remote-agent ws-proxy stop
  remote-agent ws-proxy status
  remote-agent ws-proxy config set --upstream-proxy http://squid.internal:3128
  remote-agent ws-proxy vmess-link
  remote-agent ws-proxy doctor
  remote-agent ws-proxy doctor --try-url https://example.com
  remote-agent ws-proxy sing-box client-config
  remote-agent ws-proxy sing-box run-tun --detach
  remote-agent ws-proxy vpn
`

func runWSProxy(getClient func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(wsproxyHelp)
		return nil
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "start":
		return wsproxyStart(getClient, rest)
	case "stop":
		return wsproxyStop(getClient, rest)
	case "status":
		return wsproxyStatus(getClient, rest)
	case "config":
		if len(rest) > 0 && rest[0] == "set" {
			return wsproxyConfigSet(getClient, rest[1:])
		}
		return wsproxyConfigGet(getClient, rest)
	case "vmess-link":
		return wsproxyVMessLink(getClient, rest)
	case "doctor":
		return wsproxyDoctor(getClient, rest)
	case "sing-box":
		return wsproxySingBox(getClient, rest)
	case "vpn":
		return wsproxySingBoxRunTun(getClient, rest)
	default:
		fmt.Print(wsproxyHelp)
		return nil
	}
}

func wsproxyStart(getClient func() (*client.Client, error), args []string) error {
	var tmp bool
	var dryRun bool
	var upstreamProxy string

	_, err := flags.
		Bool("--tmp", &tmp).
		Bool("--dry-run", &dryRun).
		String("--upstream-proxy", &upstreamProxy).
		Help("-h,--help", wsproxyHelp).
		Parse(args)
	if err != nil {
		return err
	}

	c, err := getClient()
	if err != nil {
		return err
	}

	if upstreamProxy != "" {
		body := fmt.Sprintf(`{"upstream_proxy":"%s"}`, upstreamProxy)
		req, err := c.NewRequest("PUT", "/api/ws-proxy/config", strings.NewReader(body))
		if err != nil {
			return err
		}
		resp, err := c.Do(req)
		if err != nil {
			return fmt.Errorf("failed to set upstream proxy: %w", err)
		}
		resp.Body.Close()
	}

	url := "/api/ws-proxy/start"
	if dryRun {
		if tmp {
			url += "?tmp=true&dry_run=true"
		} else {
			url += "?dry_run=true"
		}
	} else {
		url = "/api/ws-proxy/start/stream"
		if tmp {
			url += "?tmp=true"
		}
	}

	if dryRun {
		req, err := c.NewRequest("POST", url, strings.NewReader(""))
		if err != nil {
			return err
		}
		resp, err := c.Do(req)
		if err != nil {
			return fmt.Errorf("failed to start ws-proxy: %w", err)
		}
		defer resp.Body.Close()

		data, _ := io.ReadAll(resp.Body)
		if apiErr := parseAPIError(data); apiErr != nil {
			return fmt.Errorf("%s", mapErrorToCLI(*apiErr))
		}

		var drResult struct {
			DryRun *struct {
				Tmp           bool     `json:"tmp"`
				UpstreamProxy string   `json:"upstream_proxy"`
				ListenPort    int      `json:"listen_port"`
				WSPath        string   `json:"ws_path"`
				Subdomain     string   `json:"subdomain"`
				Domain        string   `json:"domain"`
				DomainSource  string   `json:"domain_source"`
				InstanceID    string   `json:"instance_id"`
				PublicURL     string   `json:"public_url"`
				XrayExists    bool     `json:"xray_exists"`
				XrayPath      string   `json:"xray_path"`
				PortFree      bool     `json:"port_free"`
				Checks        []string `json:"checks"`
				Issues        []string `json:"issues"`
			} `json:"dry_run"`
		}
		json.Unmarshal(data, &drResult)

		if drResult.DryRun == nil {
			return fmt.Errorf("server did not return dry_run result")
		}

		dr := drResult.DryRun

		fmt.Println("=== WS Proxy Dry Run ===")
		fmt.Printf("Mode:             %s\n", map[bool]string{true: "temporary (Quick Tunnel)", false: "permanent (user domain)"}[dr.Tmp])
		fmt.Println()
		fmt.Println("Configuration:")
		fmt.Printf("  Upstream proxy: %s\n", dr.UpstreamProxy)
		fmt.Printf("  Listen port:    %d\n", dr.ListenPort)
		fmt.Printf("  WS path:        %s\n", dr.WSPath)
		fmt.Printf("  Subdomain:      %s\n", dr.Subdomain)
		if dr.InstanceID != "" {
			fmt.Printf("  Instance ID:    %s\n", dr.InstanceID)
		}
		if dr.Domain != "" {
			fmt.Printf("  Domain:         %s\n", dr.Domain)
			if dr.DomainSource != "" {
				fmt.Printf("  Domain source:  %s\n", dr.DomainSource)
			}
		}
		if dr.PublicURL != "" {
			fmt.Printf("  Public URL:     %s\n", dr.PublicURL)
		}
		fmt.Println()
		fmt.Println("Checks:")
		for _, check := range dr.Checks {
			fmt.Printf("  ✓ %s\n", check)
		}
		if len(dr.Issues) > 0 {
			fmt.Println()
			fmt.Println("Issues (would prevent start):")
			for _, issue := range dr.Issues {
				fmt.Printf("  ✗ %s\n", issue)
			}
		}
		fmt.Println()
		if len(dr.Issues) == 0 {
			fmt.Println("All checks passed. Ready to start without --dry-run.")
		} else {
			fmt.Println("Fix the issues above before starting.")
		}
		fmt.Println("No changes were made.")
		return nil
	}

	// Streaming mode
	result, err := c.StreamSSEWithDone(url, nil, func(ev client.ServerStreamEvent) {
		if ev.Type == "log" && ev.Message != "" {
			fmt.Printf("  %s\n", ev.Message)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to start ws-proxy: %w", err)
	}
	if result == nil || result.Done == nil {
		return fmt.Errorf("stream ended without result")
	}

	publicURL, _ := result.Done["public_url"].(string)
	vmessLink, _ := result.Done["vmess_link"].(string)

	if publicURL == "" {
		return fmt.Errorf("stream ended without public_url")
	}

	if tmp {
		fmt.Println("=== Temporary WS Proxy ===")
		fmt.Printf("Public URL:  %s\n", publicURL)
		fmt.Printf("VMess Link:  %s\n", vmessLink)
		fmt.Println()
		fmt.Println("Import the vmess:// link into Shadowrocket, or configure manually:")
		fmt.Println("  Type: VMess")
		fmt.Printf("  Address: %s\n", strings.TrimPrefix(strings.TrimPrefix(publicURL, "https://"), "http://"))
		fmt.Println("  Port: 443")
		fmt.Println("  Network: ws")
		fmt.Println("  Path: /ws")
		fmt.Println("  TLS: ON")
		fmt.Println()
		fmt.Println("Press Ctrl-C to stop and discard...")

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		fmt.Println("\nStopping...")
		stopReq, err := c.NewRequest("POST", "/api/ws-proxy/stop", nil)
		if err != nil {
			return fmt.Errorf("failed to stop: %w", err)
		}
		stopResp, err := c.Do(stopReq)
		if err != nil {
			return fmt.Errorf("failed to stop: %w", err)
		}
		stopResp.Body.Close()
		fmt.Println("Stopped.")
		return nil
	}

	fmt.Printf("WS Proxy started: %s\n", publicURL)
	fmt.Printf("VMess Link: %s\n", vmessLink)
	return nil
}

func wsproxyStop(getClient func() (*client.Client, error), args []string) error {
	c, err := getClient()
	if err != nil {
		return err
	}

	req, err := c.NewRequest("POST", "/api/ws-proxy/stop", nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop ws-proxy: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return fmt.Errorf("failed to stop: %s", apiErr.Message)
	}

	fmt.Println("WS Proxy stopped.")
	return nil
}

func wsproxyStatus(getClient func() (*client.Client, error), args []string) error {
	c, err := getClient()
	if err != nil {
		return err
	}

	req, err := c.NewRequest("GET", "/api/ws-proxy/status", nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return fmt.Errorf("failed to get status: %s", apiErr.Message)
	}

	var status struct {
		Running   bool   `json:"running"`
		PublicURL string `json:"public_url"`
		Port      int    `json:"port"`
		IsTmp     bool   `json:"is_tmp"`
	}
	json.Unmarshal(data, &status)

	fmt.Printf("Running:    %v\n", status.Running)
	fmt.Printf("Public URL: %s\n", status.PublicURL)
	fmt.Printf("Port:       %d\n", status.Port)
	fmt.Printf("Temporary:  %v\n", status.IsTmp)
	return nil
}

func wsproxyConfigGet(getClient func() (*client.Client, error), args []string) error {
	c, err := getClient()
	if err != nil {
		return err
	}

	req, err := c.NewRequest("GET", "/api/ws-proxy/config", nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return fmt.Errorf("failed to get config: %s", apiErr.Message)
	}

	fmt.Println(string(data))
	return nil
}

func wsproxyConfigSet(getClient func() (*client.Client, error), args []string) error {
	var upstreamProxy string
	var listenPort int
	var wsPath string
	var subdomain string
	var autoStart bool

	_, err := flags.
		String("--upstream-proxy", &upstreamProxy).
		Int("--port", &listenPort).
		String("--path", &wsPath).
		String("--subdomain", &subdomain).
		Bool("--auto-start", &autoStart).
		Help("-h,--help", wsproxyHelp).
		Parse(args)
	if err != nil {
		return err
	}

	c, err := getClient()
	if err != nil {
		return err
	}

	body := map[string]interface{}{}
	if upstreamProxy != "" {
		body["upstream_proxy"] = upstreamProxy
	}
	if listenPort > 0 {
		body["listen_port"] = listenPort
	}
	if wsPath != "" {
		body["ws_path"] = wsPath
	}
	if subdomain != "" {
		body["subdomain"] = subdomain
	}
	if argsHave("--auto-start", args) {
		body["auto_start"] = autoStart
	}

	bodyData, _ := json.Marshal(body)
	req, err := c.NewRequest("PUT", "/api/ws-proxy/config", strings.NewReader(string(bodyData)))
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return fmt.Errorf("failed to update config: %s", apiErr.Message)
	}

	fmt.Println(string(data))
	return nil
}

const vmessLinkHelp = `Usage: remote-agent ws-proxy vmess-link [--export FILE] [--smaller-qr]

  Get the vmess:// link and manual config for import into supported clients
  (Shadowrocket, V2RayU, Clash, sing-box, v2rayNG, Surge, etc.).
  Always displays a QR code for phone scanning.

Options:
  --export FILE    write output to FILE
  --smaller-qr     use compact quadrant QR (N/2 x N/2); default is full-size
  -h, --help       show this help
`

func wsproxyVMessLink(getClient func() (*client.Client, error), args []string) error {
	var exportFile string
	var smallerQR bool

	_, err := flags.
		String("--export", &exportFile).
		Bool("--smaller-qr", &smallerQR).
		Help("-h,--help", vmessLinkHelp).
		Parse(args)
	if err != nil {
		return err
	}

	c, err := getClient()
	if err != nil {
		return err
	}

	req, err := c.NewRequest("GET", "/api/ws-proxy/vmess-link", nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get vmess link: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if apiErr := parseAPIError(data); apiErr != nil {
		return fmt.Errorf("%s", mapErrorToCLI(*apiErr))
	}

	var result struct {
		VMessLink string `json:"vmess_link"`
		Host      string `json:"host"`
		Port      string `json:"port"`
		UUID      string `json:"uuid"`
		AlterID   string `json:"alter_id"`
		Network   string `json:"network"`
		Type      string `json:"type"`
		Path      string `json:"path"`
		TLS       string `json:"tls"`
	}
	json.Unmarshal(data, &result)

	qrStr := generateQRCode(result.VMessLink, smallerQR)

	var buf strings.Builder

	buf.WriteString(result.VMessLink)
	buf.WriteString("\n\n")
	buf.WriteString("── Import ─────────────────────────────────────────\n")
	buf.WriteString("  iOS / iPadOS\n")
	buf.WriteString("    Shadowrocket:     Scan QR code or paste link in Safari\n")
	buf.WriteString("    Quantumult X:     Import vmess:// link\n")
	buf.WriteString("    Surge:            Import vmess:// link\n")
	buf.WriteString("\n")
	buf.WriteString("  macOS\n")
	buf.WriteString("    V2RayU:           Paste vmess:// link\n")
	buf.WriteString("    Shadowrocket:     Paste vmess:// link   (M-chip)\n")
	buf.WriteString("    Surge:            Import vmess:// link\n")
	buf.WriteString("    Clash / sing-box: Manual config below\n")
	buf.WriteString("\n")
	buf.WriteString("  Android\n")
	buf.WriteString("    v2rayNG:          Paste vmess:// link\n")
	buf.WriteString("\n")
	buf.WriteString("── Config ────────────────────────────────────────\n")
	buf.WriteString(fmt.Sprintf("  Type:    VMess\n"))
	buf.WriteString(fmt.Sprintf("  Address: %s\n", result.Host))
	buf.WriteString(fmt.Sprintf("  Port:    %s\n", result.Port))
	buf.WriteString(fmt.Sprintf("  UUID:    %s\n", result.UUID))
	buf.WriteString(fmt.Sprintf("  AlterId: %s\n", result.AlterID))
	buf.WriteString(fmt.Sprintf("  Network: %s\n", result.Network))
	buf.WriteString(fmt.Sprintf("  Path:    %s\n", result.Path))
	buf.WriteString(fmt.Sprintf("  TLS:     %s\n", result.TLS))
	buf.WriteString("\n")
	buf.WriteString("  Tip: --export FILE to save output\n")
	buf.WriteString("\n")
	buf.WriteString(qrStr)

	output := buf.String()
	fmt.Print(output)

	if exportFile != "" {
		if err := os.WriteFile(exportFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write export file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\nExported to: %s\n", exportFile)
	}

	return nil
}

func generateQRCode(content string, smaller bool) string {
	qr, err := qrcode.New(content, qrcode.Low)
	if err != nil {
		return fmt.Sprintf("[QR code error: %v]", err)
	}
	if smaller {
		return renderQuadrantQR(qr)
	}
	return qr.ToSmallString(false)
}

func renderQuadrantQR(qr *qrcode.QRCode) string {
	bmp := qr.Bitmap()
	return renderQuadrantQRFromBitmap(bmp)
}

func renderQuadrantQRFromBitmap(bmp [][]bool) string {
	nrow := len(bmp)
	if nrow == 0 {
		return ""
	}
	ncol := len(bmp[0])

	quietZone := 4
	if nrow <= 2*quietZone || ncol <= 2*quietZone {
		quietZone = 0
	}
	startY := quietZone
	endY := nrow - quietZone
	startX := quietZone
	endX := ncol - quietZone

	var buf strings.Builder
	for y := startY; y < endY; y += 2 {
		for x := startX; x < endX; x++ {
			ul := bmp[y][x]
			ur := false
			if x+1 < endX {
				ur = bmp[y][x+1]
			}
			ll := false
			lr := false
			if y+1 < endY {
				ll = bmp[y+1][x]
				if x+1 < endX {
					lr = bmp[y+1][x+1]
				}
			}

			var idx int
			if ul {
				idx |= 8
			}
			if ur {
				idx |= 4
			}
			if ll {
				idx |= 2
			}
			if lr {
				idx |= 1
			}

			buf.WriteRune(quadrantChars[idx])

			if x+1 < endX {
				x++
			}
		}
		buf.WriteRune('\n')
	}
	return buf.String()
}

var quadrantChars = [16]rune{
	' ', '▗', '▖', '▄',
	'▝', '▐', '▞', '▟',
	'▘', '▚', '▌', '▙',
	'▀', '▜', '▛', '█',
}

func argsHave(name string, args []string) bool {
	for _, a := range args {
		if a == name || strings.HasPrefix(a, name+"=") {
			return true
		}
	}
	return false
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func parseAPIError(data []byte) *apiError {
	var resp struct {
		Error *apiError `json:"error"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil
	}
	if resp.Error == nil || resp.Error.Code == "" {
		return nil
	}
	return resp.Error
}

func mapErrorToCLI(e apiError) string {
	msg := e.Message
	switch e.Code {
	case "NOT_CONFIGURED":
		return fmt.Sprintf("%s\n  Run: remote-agent ws-proxy config set --upstream-proxy URL", msg)
	case "ALREADY_RUNNING":
		return fmt.Sprintf("%s\n  Run: remote-agent ws-proxy stop  (to restart)", msg)
	case "NO_DOMAIN":
		return fmt.Sprintf("%s\n  Run: remote-agent ws-proxy start --tmp  (for temporary Quick Tunnel)", msg)
	case "NOT_RUNNING":
		return fmt.Sprintf("%s\n  Run: remote-agent ws-proxy start  (to start the proxy)", msg)
	default:
		return fmt.Sprintf("server error [%s]: %s", e.Code, msg)
	}
}
