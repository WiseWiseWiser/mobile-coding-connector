package wsproxy_singbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xhd2015/ai-critic/client"
)

// RunHttpOnlyOptions configures vpn-http-only.
type RunHttpOnlyOptions struct {
	ConfigFile string
	Yes        bool
	NoInstall  bool
	Detach     bool
	Policy     *DomainPolicy
	DNSHijack  bool
}

type httpOnlyRunBundle struct {
	configPath    string
	sidecar       *XraySidecar
	cleanupConfig func()
	stopSidecar   func()
	stopHealth    func()
}

// RunHttpOnly starts vpn-http-only (HTTP/HTTPS TUN with fallback direct).
func RunHttpOnly(getClient func() (*client.Client, error), opts RunHttpOnlyOptions) error {
	// --dns-hijack needs macOS system DNS pointed at the TUN (172.19.0.2) so libc
	// resolvers hit sing-box fakeip instead of polluted hotspot DNS (172.20.10.1).
	if !opts.DNSHijack {
		SetSkipPlatformTunDNS(true)
		defer SetSkipPlatformTunDNS(false)
	}

	if err := restoreStuckTunDNS(); err != nil {
		return err
	}

	if opts.ConfigFile == "" {
		MaybeWarnDNSPollution(opts.DNSHijack)
	}

	bundle, err := prepareHttpOnlyRun(getClient, opts)
	if err != nil {
		return err
	}

	if _, err := currentHooks.LookPath("sing-box"); err != nil {
		if opts.NoInstall {
			return fmt.Errorf("sing-box not installed (--no-install set)")
		}
		if !currentHooks.IsTTY() {
			return fmt.Errorf("sing-box not installed; install it with: %s", BrewInstallSingBoxCmd)
		}
		fmt.Println("sing-box is not installed.")
		PrintCommand(BrewInstallSingBoxCmd)
		shouldInstall := opts.Yes
		if !shouldInstall {
			confirmed := currentHooks.Confirm("Install via Homebrew? [y/N] ")
			if !confirmed {
				return fmt.Errorf("sing-box install declined")
			}
		}
		if err := currentHooks.BrewInstall(); err != nil {
			return fmt.Errorf("brew install sing-box failed: %w", err)
		}
	}

	euid := currentHooks.Geteuid()
	needSudo := euid != 0

	if opts.Detach {
		if bundle.cleanupConfig != nil {
			defer bundle.cleanupConfig()
		}
		return runHttpOnlyDetach(bundle.configPath, bundle.sidecar, needSudo)
	}

	if bundle.stopSidecar != nil {
		defer bundle.stopSidecar()
	}
	if bundle.cleanupConfig != nil {
		defer bundle.cleanupConfig()
	}
	if bundle.stopHealth != nil {
		defer bundle.stopHealth()
	}

	if needSudo && !currentHooks.IsTTY() {
		return fmt.Errorf("sing-box needs root privileges; run with sudo or from a TTY")
	}

	if hasProxyEnv() {
		fmt.Println("Note: HTTP/SOCKS proxy env vars are cleared for sing-box (ws-proxy must be reached directly).")
	}
	if systemProxyEnabled() {
		fmt.Println("Note: macOS system HTTP/HTTPS/SOCKS proxy will be disabled while the TUN is up.")
	}

	if bundle.sidecar != nil {
		port := bundle.sidecar.Port
		verifyLocalProxyAfterTun = func() error {
			return verifyXrayHTTPProxy(context.Background(), port, 10*time.Second)
		}
		defer func() { verifyLocalProxyAfterTun = nil }()

		stopHealth := StartWebOutboundHealthMonitor(port)
		bundle.stopHealth = stopHealth
		defer stopHealth()
	}

	ctx := context.Background()
	return currentHooks.RunSingBox(ctx, needSudo, bundle.configPath)
}

func prepareHttpOnlyRun(getClient func() (*client.Client, error), opts RunHttpOnlyOptions) (*httpOnlyRunBundle, error) {
	if opts.ConfigFile != "" {
		return &httpOnlyRunBundle{configPath: opts.ConfigFile}, nil
	}

	fmt.Println("Fetching VMess link from server...")
	c, err := getClient()
	if err != nil {
		return nil, err
	}
	vmess, err := currentHooks.FetchVMess(c)
	if err != nil {
		return nil, err
	}
	if proxyIPs := resolveHostIPv4CIDRs(vmess.Host); len(proxyIPs) == 0 {
		fmt.Fprintf(os.Stderr, "warning: could not resolve %s for TUN route exclusions\n", vmess.Host)
	}

	fmt.Println("Starting local xray VMess client (ws-proxy doctor path)...")
	sidecar, err := StartXraySidecar(context.Background(), vmess)
	if err != nil {
		return nil, fmt.Errorf("xray sidecar: %w", err)
	}
	fmt.Printf("xray SOCKS ready on 127.0.0.1:%d (VMess via %s)\n", sidecar.Port, vmess.Host)

	initialUseProxy := ProbeUpstreamProxy(sidecar.Port)
	if !initialUseProxy {
		fmt.Println("Upstream xray SOCKS unreachable; starting in direct-fallback mode.")
	}

	fmt.Println("Building sing-box HTTP-only TUN config...")
	data, err := BuildSingBoxHttpOnlyTunConfig(vmess, &HttpOnlyConfigOptions{
		LocalSocksPort:  sidecar.Port,
		InitialUseProxy: initialUseProxy,
		Policy:          opts.Policy,
		DNSHijack:       opts.DNSHijack,
	})
	if err != nil {
		sidecar.Stop()
		return nil, err
	}
	tmpFile, err := os.CreateTemp("", "singbox-http-only-*.json")
	if err != nil {
		sidecar.Stop()
		return nil, fmt.Errorf("create temp config: %w", err)
	}
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		sidecar.Stop()
		return nil, fmt.Errorf("write temp config: %w", err)
	}
	tmpFile.Close()

	configPath := tmpFile.Name()
	return &httpOnlyRunBundle{
		configPath: configPath,
		sidecar:    sidecar,
		cleanupConfig: func() {
			_ = os.Remove(configPath)
		},
		stopSidecar: func() {
			sidecar.Stop()
		},
	}, nil
}

func runHttpOnlyDetach(configPath string, sidecar *XraySidecar, needSudo bool) error {
	cacheDir, err := currentHooks.UserCacheDir()
	if err != nil {
		return fmt.Errorf("cache dir: %w", err)
	}
	singBoxDir := filepath.Join(cacheDir, "singbox")
	if err := os.MkdirAll(singBoxDir, 0700); err != nil {
		return fmt.Errorf("create singbox dir: %w", err)
	}
	runConfigPath := filepath.Join(singBoxDir, "http-only-run.json")
	logPath := filepath.Join(singBoxDir, "sing-box-http-only.log")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := os.WriteFile(runConfigPath, data, 0600); err != nil {
		return fmt.Errorf("write run config: %w", err)
	}

	pid, err := currentHooks.StartDetached(runConfigPath, logPath, needSudo)
	if err != nil {
		return fmt.Errorf("start detached: %w", err)
	}

	fmt.Printf("sing-box HTTP-only started in background (PID: %d)\n", pid)
	fmt.Printf("Config: %s\n", runConfigPath)
	fmt.Printf("Log:    %s\n", logPath)
	if sidecar != nil && sidecar.cmd != nil && sidecar.cmd.Process != nil {
		xrayPIDPath := filepath.Join(singBoxDir, "xray-http-only.pid")
		_ = os.WriteFile(xrayPIDPath, []byte(strconv.Itoa(sidecar.cmd.Process.Pid)), 0600)
		fmt.Printf("xray sidecar (PID: %d, SOCKS 127.0.0.1:%d)\n", sidecar.cmd.Process.Pid, sidecar.Port)
		fmt.Printf("xray PID file: %s\n", xrayPIDPath)
		fmt.Println("Note: upstream health monitoring runs in foreground mode only; restart vpn-http-only if ws-proxy flaps while detached.")
	}
	return nil
}