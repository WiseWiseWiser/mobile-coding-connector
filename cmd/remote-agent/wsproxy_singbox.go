package main

import (
	"fmt"

	"github.com/xhd2015/ai-critic/client"
	singbox "github.com/xhd2015/ai-critic/cmd/remote-agent/wsproxy_singbox"
	"github.com/xhd2015/less-gen/flags"
)

const singBoxHelp = `Usage: remote-agent ws-proxy sing-box <subcommand> [options]

Manage sing-box TUN client for ws-proxy.

Subcommands:
  client-config [--output FILE]
      Fetch VMess params from server and emit a sing-box TUN config.
      Default writes JSON to stdout.

  run-tun [--yes] [--no-install] [--config FILE] [--detach]
      Start sing-box TUN tunnel.
      Without --config, fetches VMess params and builds config automatically.
      With --detach, starts sing-box in background and exits.

Options:
  --yes          Skip install confirmation prompt (implies yes to Homebrew)
  --no-install   Fail if sing-box is not on PATH
  --config FILE  Use existing sing-box config file
  --detach       Run sing-box in background

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
	var yes bool
	var noInstall bool
	var configFile string
	var detach bool
	_, err := flags.
		Bool("--yes", &yes).
		Bool("--no-install", &noInstall).
		String("--config", &configFile).
		Bool("--detach", &detach).
		Help("-h,--help", singBoxHelp).
		Parse(args)
	if err != nil {
		return err
	}
	return singbox.RunTun(getClient, singbox.RunTunOptions{
		ConfigFile: configFile,
		Yes:        yes,
		NoInstall:  noInstall,
		Detach:     detach,
	})
}
