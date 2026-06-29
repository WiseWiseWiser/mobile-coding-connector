package main

import (
	"github.com/xhd2015/ai-critic/client"
	singbox "github.com/xhd2015/ai-critic/cmd/remote-agent/wsproxy_singbox"
	"github.com/xhd2015/less-gen/flags"
)

const httpOnlyHelp = `Usage: remote-agent ws-proxy vpn-http-only [options]

Start a lightweight HTTP/HTTPS-only TUN tunnel via ws-proxy.
No HTTP_PROXY env vars are required. Non-web traffic stays direct.
When the ws-proxy path is unavailable, HTTP/HTTPS falls back to direct.

Options:
  --whitelist              Only proxy domains matched by --include
  --blacklist              Proxy HTTP/HTTPS except --exclude (default when no patterns)
  --include PATTERN        Repeatable; exact.com or *.zone patterns
  --exclude PATTERN        Repeatable; holes within the active mode
  --dns-hijack             Hijack DNS via TUN fakeip + set system DNS (fixes polluted hotspot DNS)
  --yes                    Skip sing-box install confirmation
  --no-install             Fail if sing-box is not on PATH
  --config FILE            Use an existing sing-box config file
  --detach                 Run sing-box in background

Mode inference:
  only --include       -> whitelist
  only --exclude       -> blacklist
  both lists           -> require --whitelist or --blacklist
  no patterns          -> blacklist (proxy all HTTP/HTTPS)

Examples:
  remote-agent ws-proxy vpn-http-only
  remote-agent ws-proxy vpn-http-only --dns-hijack
  remote-agent ws-proxy vpn-http-only --whitelist --include '*.internal.corp'
  remote-agent ws-proxy vpn-http-only --blacklist --exclude github.com --exclude '*.google.com'
  remote-agent ws-proxy vpn-http-only --whitelist \
    --include '*.corp.com' --exclude 'cdn.corp.com'
`

func wsproxyHttpOnly(getClient func() (*client.Client, error), args []string) error {
	var yes bool
	var noInstall bool
	var configFile string
	var detach bool
	var whitelist bool
	var blacklist bool
	var includes []string
	var excludes []string
	var dnsHijack bool

	_, err := flags.
		Bool("--dns-hijack", &dnsHijack).
		Bool("--yes", &yes).
		Bool("--no-install", &noInstall).
		String("--config", &configFile).
		Bool("--detach", &detach).
		Bool("--whitelist", &whitelist).
		Bool("--blacklist", &blacklist).
		StringSlice("--include", &includes).
		StringSlice("--exclude", &excludes).
		Help("-h,--help", httpOnlyHelp).
		Parse(args)
	if err != nil {
		return err
	}

	policy, err := singbox.ParseDomainPolicy(singbox.PolicyInput{
		Whitelist: whitelist,
		Blacklist: blacklist,
		Include:   includes,
		Exclude:   excludes,
	})
	if err != nil {
		return err
	}

	return singbox.RunHttpOnly(getClient, singbox.RunHttpOnlyOptions{
		ConfigFile: configFile,
		Yes:        yes,
		NoInstall:  noInstall,
		Detach:     detach,
		Policy:     policy,
		DNSHijack:  dnsHijack,
	})
}