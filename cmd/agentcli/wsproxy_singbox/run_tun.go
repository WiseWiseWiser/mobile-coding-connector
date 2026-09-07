package wsproxy_singbox

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/dot-pkgs/go-pkgs/singboxtun"
)

const (
	tunCacheDirName  = "remote-agent"
	tunSudoersName   = "remote-agent-sing-box"
	tunDNSHijackHint = "Retry with: remote-agent ws-proxy vpn --http-only --dns-hijack"
)

// RunTun starts sing-box TUN for ws-proxy (full VPN or --http-only mode).
func RunTun(getClient func() (*client.Client, error), opts RunTunOptions) error {
	if len(opts.RemoteDirect) > 0 {
		if err := pushRemoteDirect(getClient, opts.RemoteDirect); err != nil {
			return err
		}
	}

	restoreHooks := bridgeHooksToSingboxtun()
	defer restoreHooks()

	if opts.ConfigFile != "" {
		return singboxtun.RunTun(toSingboxtunOptions(opts, 0, ""))
	}

	fmt.Println("Fetching VMess link from server...")
	c, err := getClient()
	if err != nil {
		return err
	}
	vmess, err := currentHooks.FetchVMess(c)
	if err != nil {
		return err
	}

	fmt.Println("Starting local xray VMess client (ws-proxy doctor path)...")
	sidecar, err := StartXraySidecar(context.Background(), vmess)
	if err != nil {
		return fmt.Errorf("xray sidecar: %w", err)
	}
	fmt.Printf("xray SOCKS ready on 127.0.0.1:%d (VMess via %s)\n", sidecar.Port, vmess.Host)

	if !opts.Detach {
		defer sidecar.Stop()
	}

	if !opts.Detach {
		port := sidecar.Port
		verifyLocalProxyAfterTun = func() error {
			return verifyXrayHTTPProxy(context.Background(), port, 10*time.Second)
		}
		defer func() { verifyLocalProxyAfterTun = nil }()
	}

	err = singboxtun.RunTun(toSingboxtunOptions(opts, sidecar.Port, vmess.Host))
	if opts.Detach && err == nil {
		writeDetachedXrayPID(sidecar, opts.HttpOnly)
	}
	if opts.Detach && err != nil {
		sidecar.Stop()
	}
	return err
}

// RunHttpOnly is deprecated; use RunTun with HttpOnly set.
func RunHttpOnly(getClient func() (*client.Client, error), opts RunHttpOnlyOptions) error {
	opts.HttpOnly = true
	return RunTun(getClient, opts)
}

func toSingboxtunOptions(opts RunTunOptions, localSocksPort int, proxyHost string) singboxtun.RunTunOptions {
	return singboxtun.RunTunOptions{
		LocalSocksPort: localSocksPort,
		ProxyHost:      proxyHost,
		ConfigFile:     opts.ConfigFile,
		Yes:            opts.Yes,
		NoInstall:      opts.NoInstall,
		NoSetupSudo:    opts.NoSetupSudo,
		Detach:         opts.Detach,
		HttpOnly:       opts.HttpOnly,
		DNSHijack:      opts.DNSHijack,
		Policy:         toSingboxtunPolicy(opts.Policy),
		AlsoProxy:      toSingboxtunAlsoProxy(opts.AlsoProxy),
		CacheDirName:   tunCacheDirName,
		SudoersName:    tunSudoersName,
		DNSHijackHint:  tunDNSHijackHint,
	}
}

func toSingboxtunPolicy(p *DomainPolicy) *singboxtun.DomainPolicy {
	if p == nil {
		return nil
	}
	out := &singboxtun.DomainPolicy{Mode: singboxtun.PolicyMode(p.Mode)}
	for _, d := range p.Include {
		out.Include = append(out.Include, singboxtun.DomainPattern{
			Raw: d.Raw, Wildcard: d.Wildcard, Value: d.Value,
		})
	}
	for _, d := range p.Exclude {
		out.Exclude = append(out.Exclude, singboxtun.DomainPattern{
			Raw: d.Raw, Wildcard: d.Wildcard, Value: d.Value,
		})
	}
	return out
}

func toSingboxtunAlsoProxy(in []AlsoProxyPattern) []singboxtun.AlsoProxyPattern {
	if len(in) == 0 {
		return nil
	}
	out := make([]singboxtun.AlsoProxyPattern, len(in))
	for i, p := range in {
		out[i] = singboxtun.AlsoProxyPattern{
			Raw: p.Raw, Wildcard: p.Wildcard, Host: p.Host, Port: p.Port,
		}
	}
	return out
}

// bridgeHooksToSingboxtun forwards wsproxy test/runtime hooks into singboxtun
// so InstallTestHooks and production RunSingBox (xray verify) keep working.
func bridgeHooksToSingboxtun() func() {
	return singboxtun.InstallTestHooks(singboxtun.TestHooks{
		LookPath:    currentHooks.LookPath,
		IsTTY:       currentHooks.IsTTY,
		Confirm:     currentHooks.Confirm,
		BrewInstall: currentHooks.BrewInstall,
		Geteuid:     currentHooks.Geteuid,
		RunSingBox:  currentHooks.RunSingBox,
		StartDetached: currentHooks.StartDetached,
		UserCacheDir:  currentHooks.UserCacheDir,
		EnsureSudoSetup: func(singBoxPath string, noSetup bool, _, _ string) error {
			if currentHooks.EnsureSudoSetup == nil {
				return nil
			}
			return currentHooks.EnsureSudoSetup(singBoxPath, noSetup)
		},
	})
}

func writeDetachedXrayPID(sidecar *XraySidecar, httpOnly bool) {
	if sidecar == nil || sidecar.cmd == nil || sidecar.cmd.Process == nil {
		return
	}
	cacheDir, err := currentHooks.UserCacheDir()
	if err != nil {
		return
	}
	dir := filepath.Join(cacheDir, tunCacheDirName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return
	}
	name := "xray.pid"
	if httpOnly {
		name = "xray-http-only.pid"
	}
	path := filepath.Join(dir, name)
	_ = os.WriteFile(path, []byte(strconv.Itoa(sidecar.cmd.Process.Pid)), 0600)
	fmt.Printf("xray sidecar (PID: %d, SOCKS 127.0.0.1:%d)\n", sidecar.cmd.Process.Pid, sidecar.Port)
	fmt.Printf("xray PID file: %s\n", path)
}
