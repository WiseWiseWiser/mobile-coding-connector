package cloudflared

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/server/cloudflare"
	"github.com/xhd2015/ai-critic/server/cloudflare/unified_tunnel"
	"github.com/xhd2015/ai-critic/server/cloudflareproxy"
	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/ai-critic/server/domains"
	serverqemu "github.com/xhd2015/ai-critic/server/qemu"
	"github.com/xhd2015/ai-critic/server/services"
	sharedqemu "github.com/xhd2015/dot-pkgs/go-pkgs/qemu"
)

// Snapshot is the files plus the running connectors.
type Snapshot struct {
	Files   Files
	Runtime Runtime
}

// LoadFiles reads the on-disk selector. It does not probe the guest or the edge.
func LoadFiles() (Files, string, error) {
	cfg, err := cloudflare.LoadConfig()
	if err != nil {
		return Files{}, "", err
	}
	if cfg == nil {
		cfg = &cloudflare.CloudflareConfig{}
	}
	qcfg, err := serverqemu.LoadConfig()
	if err != nil {
		return Files{}, "", err
	}
	f := Files{
		Mode:        cfg.Mode,
		ProxyURL:    strings.TrimSpace(cfg.ProxyURL),
		HasToken:    strings.TrimSpace(cfg.Token) != "",
		QemuEnabled: qcfg.Enabled,
	}
	return f, primaryHost(), nil
}

// Observe reads the selector files and which connectors are up.
func Observe() (Snapshot, error) {
	cfg, err := cloudflare.LoadConfig()
	if err != nil {
		return Snapshot{}, err
	}
	qcfg, err := serverqemu.LoadConfig()
	if err != nil {
		return Snapshot{}, err
	}
	f := Files{
		Mode:        cfg.Mode,
		ProxyURL:    strings.TrimSpace(cfg.ProxyURL),
		HasToken:    strings.TrimSpace(cfg.Token) != "",
		QemuEnabled: qcfg.Enabled,
	}
	rt := Runtime{
		PublicHost:               primaryHost(),
		HostCloudflaredInstalled: cloudflare.IsCommandAvailable("cloudflared"),
		QemuInstalled:            qemuInstalled(),
	}
	if kv, err := serverqemu.DefaultManager().CFStatus(""); err == nil {
		rt.GuestCloudflared = kv["cf_alive"] == "yes"
		rt.GuestCFPID = kv["cf_pid"]
	}
	rt.ProxySessions = len(cloudflare.ProxySessionDomains())
	if counts, err := cloudflare.ProxyDialCounts(); err == nil {
		for _, n := range counts {
			rt.ProxyDials += n
		}
	}
	if tg := unified_tunnel.GetTunnelGroupManager().GetCoreGroup(); tg != nil && tg.IsRunning() {
		rt.HostCloudflared = true
	}
	return Snapshot{Files: f, Runtime: rt}, nil
}

func qemuInstalled() bool {
	_, err := exec.LookPath("qemu-system-x86_64")
	return err == nil
}

func primaryHost() string {
	cfg, err := domains.LoadDomains()
	if err != nil || cfg == nil {
		return ""
	}
	for _, d := range cfg.Domains {
		if d.Provider == domains.ProviderCloudflare && strings.TrimSpace(d.Domain) != "" {
			return d.Domain
		}
	}
	return ""
}

type hostRef struct {
	Host      string
	ServiceID string
}

func expectedHosts() []hostRef {
	var out []hostRef
	seen := map[string]bool{}
	if cfg, err := domains.LoadDomains(); err == nil && cfg != nil {
		for _, d := range cfg.Domains {
			host := strings.TrimSpace(d.Domain)
			if d.Provider != domains.ProviderCloudflare || host == "" || seen[host] {
				continue
			}
			seen[host] = true
			out = append(out, hostRef{Host: host})
		}
	}
	for _, h := range services.GetDefaultManager().CloudflareOwnedForwardHosts() {
		host := strings.TrimSpace(h.Host)
		if host == "" || seen[host] {
			continue
		}
		seen[host] = true
		out = append(out, hostRef{Host: host, ServiceID: h.ServiceID})
	}
	return out
}

// Apply runs a planned switch. Preflight failures return before any file write.
// A later failure leaves the new files in place so a retry is idempotent, and
// does not stop the previous connector.
func Apply(target Backend, log func(string)) (UseSummary, error) {
	if log == nil {
		log = func(string) {}
	}
	snap, err := Observe()
	if err != nil {
		return UseSummary{}, err
	}
	sw, err := PlanSwitch(snap.Files, snap.Runtime, target)
	if err != nil {
		return UseSummary{}, err
	}
	if target == BackendProxy && !sw.Noop {
		if err := probeEdge(snap.Files.ProxyURL, 10*time.Second); err != nil {
			return UseSummary{}, fmt.Errorf("edge unreachable: %w", err)
		}
	}
	if sw.Noop {
		log("already on " + string(target))
		code, err := verifyPublic()
		if err != nil {
			return UseSummary{}, err
		}
		return summaryFrom(target, snap, code, true), nil
	}

	var ping int
	for _, step := range sw.Steps {
		switch step {
		case StepWrite:
			log("write: " + string(target))
			if err := writeFiles(target); err != nil {
				return UseSummary{}, err
			}
		case StepRepublish:
			if err := republish(target, log); err != nil {
				return UseSummary{}, err
			}
		case StepVerify:
			code, err := verifyPublic()
			if err != nil {
				return UseSummary{}, err
			}
			ping = code
			log(fmt.Sprintf("ping: %d", code))
		case StepStopGuest:
			log("stop: guest cloudflared")
			if err := serverqemu.DefaultManager().CFStop(); err != nil {
				return UseSummary{}, fmt.Errorf("stop guest cloudflared: %w", err)
			}
		case StepStopProxy:
			log("stop: proxy publish")
			if err := stopAllProxy(); err != nil {
				return UseSummary{}, err
			}
		case StepStopHost:
			log("stop: host cloudflared")
			stopHostCloudflared()
		default:
			return UseSummary{}, fmt.Errorf("unknown step %s", step)
		}
	}
	after, err := Observe()
	if err != nil {
		return UseSummary{}, err
	}
	return summaryFrom(target, after, ping, false), nil
}

func summaryFrom(target Backend, snap Snapshot, ping int, noop bool) UseSummary {
	s := UseSummary{
		Target:    target,
		ProxyURL:  snap.Files.ProxyURL,
		Published: publishedLine(snap.Runtime.PublicHost),
		Ping:      ping,
	}
	if target == BackendProxy {
		if snap.Runtime.GuestCloudflared {
			s.QemuLine = "disabled  guest cloudflared still running"
		} else if noop {
			s.QemuLine = "disabled  guest cloudflared stopped"
		} else {
			s.QemuLine = "disabled  guest cloudflared stopped"
		}
	} else if snap.Files.QemuEnabled {
		s.QemuLine = qemuStatus(snap.Files, snap.Runtime)
	}
	if s.Published == "" {
		s.Published = "(none)"
	}
	return s
}

func publishedLine(primary string) string {
	hosts := expectedHosts()
	if len(hosts) == 0 {
		return primary
	}
	names := make([]string, 0, len(hosts))
	for _, h := range hosts {
		names = append(names, h.Host)
	}
	if primary != "" {
		return primary
	}
	return strings.Join(names, ",")
}

func writeFiles(target Backend) error {
	cfg, err := cloudflare.LoadConfig()
	if err != nil {
		return err
	}
	if cfg == nil {
		cfg = &cloudflare.CloudflareConfig{}
	}
	switch target {
	case BackendProxy:
		cfg.Mode = "proxy"
	default:
		cfg.Mode = ""
	}
	if err := cloudflare.SaveConfig(cfg); err != nil {
		return err
	}
	_ = os.Chmod(config.CloudflareFile, 0o600)
	return serverqemu.SaveConfig(sharedqemu.FileConfig{Enabled: target == BackendQemu})
}

func republish(target Backend, log func(string)) error {
	hosts := expectedHosts()
	if len(hosts) == 0 {
		return fmt.Errorf("no cloudflare hostnames to publish")
	}
	if target == BackendProxy {
		for _, h := range hosts {
			cloudflare.StopProxyPublish(h.Host)
		}
		waitProxyQuiet(hosts, 8*time.Second)
	}
	var failed []string
	for _, h := range hosts {
		var err error
		if h.ServiceID != "" {
			err = services.GetDefaultManager().RepublishForward(h.ServiceID)
		} else if target == BackendProxy {
			port := domains.GetServerPort()
			if port == 0 {
				port = 23712
			}
			_, err = domains.StartHostDomainTunnel(h.Host, port, log)
		} else {
			port := domains.GetServerPort()
			if port == 0 {
				port = 23712
			}
			_, err = domains.ForceStartHostDomainTunnel(h.Host, port, log)
		}
		if err != nil {
			failed = append(failed, h.Host+": "+err.Error())
			continue
		}
		log("publish: " + h.Host)
	}
	if len(failed) > 0 {
		return fmt.Errorf("%s", strings.Join(failed, "; "))
	}
	return nil
}

func waitProxyQuiet(hosts []hostRef, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		counts, err := cloudflare.ProxyDialCounts()
		if err != nil {
			return
		}
		busy := false
		for _, h := range hosts {
			if counts[h.Host] > 0 {
				busy = true
				break
			}
		}
		if !busy {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func stopAllProxy() error {
	var failed []string
	for _, h := range expectedHosts() {
		cloudflare.StopProxyPublish(h.Host)
	}
	// Sessions not in the expected set (a hostname dropped from config) too.
	for _, host := range cloudflare.ProxySessionDomains() {
		cloudflare.StopProxyPublish(host)
	}
	waitProxyQuiet(expectedHosts(), 8*time.Second)
	if n := len(cloudflare.ProxySessionDomains()); n > 0 {
		failed = append(failed, fmt.Sprintf("%d proxy sessions still registered", n))
	}
	if len(failed) > 0 {
		return fmt.Errorf("%s", strings.Join(failed, "; "))
	}
	return nil
}

func stopHostCloudflared() {
	tg := unified_tunnel.GetTunnelGroupManager().GetCoreGroup()
	if tg == nil {
		return
	}
	tg.TunnelMgr().Stop()
}

func verifyPublic() (int, error) {
	host := primaryHost()
	if host == "" {
		return 0, fmt.Errorf("no public hostname to verify")
	}
	code, err := probeURL("https://"+host+"/ping", 20*time.Second)
	if err != nil {
		return 0, fmt.Errorf("ping %s: %w", host, err)
	}
	if !pingReached(code) {
		return code, fmt.Errorf("ping %s returned %d", host, code)
	}
	return code, nil
}

func probeEdge(proxyURL string, timeout time.Duration) error {
	cfg, err := cloudflare.LoadConfig()
	if err != nil {
		return err
	}
	if cfg == nil || strings.TrimSpace(cfg.Token) == "" {
		return fmt.Errorf("token is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err = (&cloudflareproxy.APIClient{
		BaseURL: strings.TrimSpace(proxyURL),
		Token:   strings.TrimSpace(cfg.Token),
	}).ListMappingsContext(ctx)
	return err
}

func probeURL(raw string, timeout time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return 0, err
	}
	resp, err := (&http.Client{Timeout: timeout}).Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func pingReached(code int) bool {
	switch code {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, 530:
		return false
	default:
		return code > 0 && code < 500
	}
}
