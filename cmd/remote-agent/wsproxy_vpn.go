package main

import (
	"github.com/xhd2015/ai-critic/client"
	singbox "github.com/xhd2015/ai-critic/cmd/remote-agent/wsproxy_singbox"
	"github.com/xhd2015/less-gen/flags"
)

const vpnHelp = `Usage: remote-agent ws-proxy vpn [options]

Start a sing-box TUN tunnel via ws-proxy.

Default mode proxies all traffic (full mini-VPN). With --http-only, only HTTP/HTTPS
goes through ws-proxy; other traffic stays direct and ws-proxy outages fall back to direct.

Options:
  --http-only            HTTP/HTTPS-only TUN with direct fallback
  --whitelist            Only proxy domains matched by --include
  --blacklist            Proxy except --exclude (default when patterns given in --http-only)
  --include PATTERN      Repeatable; exact.com or *.zone patterns
  --exclude PATTERN      Repeatable; holes within the active mode
  --dns-hijack           Hijack DNS via TUN fakeip (default in full VPN; optional in --http-only)
  --yes                  Skip sing-box install confirmation
  --no-install           Fail if sing-box is not on PATH
  --config FILE          Use an existing sing-box config file
  --detach               Run sing-box in background

Mode inference:
  only --include       -> whitelist
  only --exclude       -> blacklist
  both lists           -> require --whitelist or --blacklist
  no patterns          -> full VPN: proxy all; --http-only: proxy all HTTP/HTTPS

Examples:
  remote-agent ws-proxy vpn
  remote-agent ws-proxy vpn --http-only --dns-hijack
  remote-agent ws-proxy vpn --blacklist --exclude github.com
  remote-agent ws-proxy vpn --http-only --whitelist --include '*.internal.corp'
`

func wsproxyVpn(getClient func() (*client.Client, error), args []string) error {
	opts, err := parseVpnFlags(args, vpnHelp)
	if err != nil {
		return err
	}
	return singbox.RunTun(getClient, opts)
}

func parseVpnFlags(args []string, help string) (singbox.RunTunOptions, error) {
	var opts singbox.RunTunOptions
	var whitelist bool
	var blacklist bool
	var includes []string
	var excludes []string

	_, err := flags.
		Bool("--http-only", &opts.HttpOnly).
		Bool("--dns-hijack", &opts.DNSHijack).
		Bool("--yes", &opts.Yes).
		Bool("--no-install", &opts.NoInstall).
		String("--config", &opts.ConfigFile).
		Bool("--detach", &opts.Detach).
		Bool("--whitelist", &whitelist).
		Bool("--blacklist", &blacklist).
		StringSlice("--include", &includes).
		StringSlice("--exclude", &excludes).
		Help("-h,--help", help).
		Parse(args)
	if err != nil {
		return opts, err
	}

	if len(includes) > 0 || len(excludes) > 0 || whitelist || blacklist {
		policy, err := singbox.ParseDomainPolicy(singbox.PolicyInput{
			Whitelist: whitelist,
			Blacklist: blacklist,
			Include:   includes,
			Exclude:   excludes,
		})
		if err != nil {
			return opts, err
		}
		opts.Policy = policy
	}
	return opts, nil
}