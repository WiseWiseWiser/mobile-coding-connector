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

type tunRunBundle struct {
	configPath    string
	sidecar       *XraySidecar
	cleanupConfig func()
	stopSidecar   func()
	stopHealth    func()
}

// RunTun starts sing-box TUN for ws-proxy (full VPN or --http-only mode).
func RunTun(getClient func() (*client.Client, error), opts RunTunOptions) error {
	if opts.HttpOnly && !opts.DNSHijack {
		SetSkipPlatformTunDNS(true)
		defer SetSkipPlatformTunDNS(false)
	}

	if err := restoreStuckTunDNS(); err != nil {
		return err
	}

	if opts.ConfigFile == "" && opts.HttpOnly {
		MaybeWarnDNSPollution(opts.DNSHijack)
	}

	bundle, err := prepareTunRun(getClient, opts)
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
		return runDetach(bundle.configPath, bundle.sidecar, needSudo, opts.HttpOnly)
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

		if opts.HttpOnly {
			stopHealth := StartWebOutboundHealthMonitor(port)
			bundle.stopHealth = stopHealth
			defer stopHealth()
		}
	}

	ctx := context.Background()
	return currentHooks.RunSingBox(ctx, needSudo, bundle.configPath)
}

// RunHttpOnly is deprecated; use RunTun with HttpOnly set.
func RunHttpOnly(getClient func() (*client.Client, error), opts RunHttpOnlyOptions) error {
	opts.HttpOnly = true
	return RunTun(getClient, opts)
}

func prepareTunRun(getClient func() (*client.Client, error), opts RunTunOptions) (*tunRunBundle, error) {
	if opts.ConfigFile != "" {
		return &tunRunBundle{configPath: opts.ConfigFile}, nil
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

	buildOpts := buildTunConfigOptions(sidecar.Port)
	buildOpts.HttpOnly = opts.HttpOnly
	buildOpts.Policy = opts.Policy
	buildOpts.DNSHijack = opts.DNSHijack
	if opts.HttpOnly {
		buildOpts.InitialUseProxy = ProbeUpstreamProxy(sidecar.Port)
		if !buildOpts.InitialUseProxy {
			fmt.Println("Upstream xray SOCKS unreachable; starting in direct-fallback mode.")
		}
	}

	if opts.HttpOnly {
		fmt.Println("Building sing-box HTTP-only TUN config...")
	} else {
		fmt.Println("Building sing-box TUN config...")
	}
	data, err := BuildSingBoxTunConfig(vmess, buildOpts)
	if err != nil {
		sidecar.Stop()
		return nil, err
	}

	prefix := "singbox-"
	if opts.HttpOnly {
		prefix = "singbox-http-only-"
	}
	tmpFile, err := os.CreateTemp("", prefix+"*.json")
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
	return &tunRunBundle{
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

func runDetach(configPath string, sidecar *XraySidecar, needSudo bool, httpOnly bool) error {
	cacheDir, err := currentHooks.UserCacheDir()
	if err != nil {
		return fmt.Errorf("cache dir: %w", err)
	}
	singBoxDir := filepath.Join(cacheDir, "singbox")
	if err := os.MkdirAll(singBoxDir, 0700); err != nil {
		return fmt.Errorf("create singbox dir: %w", err)
	}

	runConfigPath := filepath.Join(singBoxDir, "run.json")
	logPath := filepath.Join(singBoxDir, "sing-box.log")
	xrayPIDPath := filepath.Join(singBoxDir, "xray.pid")
	if httpOnly {
		runConfigPath = filepath.Join(singBoxDir, "http-only-run.json")
		logPath = filepath.Join(singBoxDir, "sing-box-http-only.log")
		xrayPIDPath = filepath.Join(singBoxDir, "xray-http-only.pid")
	}

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

	modeLabel := "VPN"
	if httpOnly {
		modeLabel = "HTTP-only"
	}
	fmt.Printf("sing-box %s started in background (PID: %d)\n", modeLabel, pid)
	fmt.Printf("Config: %s\n", runConfigPath)
	fmt.Printf("Log:    %s\n", logPath)
	if sidecar != nil && sidecar.cmd != nil && sidecar.cmd.Process != nil {
		_ = os.WriteFile(xrayPIDPath, []byte(strconv.Itoa(sidecar.cmd.Process.Pid)), 0600)
		fmt.Printf("xray sidecar (PID: %d, SOCKS 127.0.0.1:%d)\n", sidecar.cmd.Process.Pid, sidecar.Port)
		fmt.Printf("xray PID file: %s\n", xrayPIDPath)
		if httpOnly {
			fmt.Println("Note: upstream health monitoring runs in foreground mode only; restart with --http-only if ws-proxy flaps while detached.")
		}
	}
	return nil
}