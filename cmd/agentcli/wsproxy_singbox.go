package agentcli

import (
	"fmt"

	"github.com/xhd2015/ai-critic/client"
	singbox "github.com/xhd2015/ai-critic/cmd/agentcli/wsproxy_singbox"
	"github.com/xhd2015/less-gen/flags"
)

const singBoxHelp = `Usage: remote-agent ws-proxy sing-box <subcommand> [options]

Manage sing-box TUN client for ws-proxy.

Subcommands:
  client-config [--output FILE]
      Fetch VMess params from server and emit a sing-box TUN config.
      Default writes JSON to stdout.

  run-tun [vpn options]
      Same as ws-proxy vpn — start sing-box TUN tunnel.

Options:
  --http-only, --whitelist, --blacklist, --include, --exclude, --dns-hijack
  --yes, --no-install, --config FILE, --detach

Examples:
  remote-agent ws-proxy sing-box client-config
  remote-agent ws-proxy sing-box client-config --output /tmp/singbox.json
  remote-agent ws-proxy sing-box run-tun
  remote-agent ws-proxy sing-box run-tun --config /tmp/singbox.json
  remote-agent ws-proxy sing-box run-tun --detach
`

func wsproxySingBox(getClient func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(singBoxHelp)
		return nil
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "client-config":
		return wsproxySingBoxClientConfig(getClient, rest)
	case "run-tun":
		return wsproxySingBoxRunTun(getClient, rest)
	default:
		fmt.Print(singBoxHelp)
		return nil
	}
}

func wsproxySingBoxClientConfig(getClient func() (*client.Client, error), args []string) error {
	var outputFile string
	_, err := flags.
		String("--output", &outputFile).
		Help("-h,--help", singBoxHelp).
		Parse(args)
	if err != nil {
		return err
	}
	return singbox.RunClientConfig(getClient, singbox.ClientConfigOptions{
		OutputFile: outputFile,
	})
}

func wsproxySingBoxRunTun(getClient func() (*client.Client, error), args []string) error {
	opts, err := parseVpnFlags(args, singBoxHelp)
	if err != nil {
		return err
	}
	return singbox.RunTun(getClient, opts)
}
