package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	remoteagentskill "github.com/xhd2015/ai-critic/cmd/remote-agent/skill"
	"github.com/xhd2015/less-gen/flags"
)

const help = `Usage: remote-agent [--server URL] [--token TOKEN] <command> [args...]

CLI for the ai-critic server API. Configure a default server with 'config';
then most commands use saved credentials.

Global options:
  --server URL    Server base URL (default: saved domain from 'config')
  --token TOKEN   Bearer token (default: saved token for --server)
  -h, --help      Show this help

Getting started:
  remote-agent config          # pick default server + token
  remote-agent ping            # check reachability
  remote-agent auth status     # verify token

Commands:
  Connectivity
    ping                 Check server reachability
    auth                 Authentication checks

  Files
    upload               Upload a local file to the server
    download             Download a remote file

  Remote shell
    exec                 Run a command on the server
    bash                 Interactive remote shell
    terminal             Persistent terminal sessions

  Git & projects
    git                  Server-side git operations
    project              Project metadata and git identity
    settings             Server settings (git users, etc.)

  Operations
    service              Managed services (start/stop/logs/upgrade)
    server               Server lifecycle (build-next, restart, status)
    proxy                Configured HTTP proxies
    request              Call arbitrary API paths

  Integrations
    agent                Custom agents and sessions
    openclaw             Mock OpenClaw gateway + Slack config
    ws-proxy             Mobile WebSocket proxy (Xray + tunnel)

  Local & tooling
    local                Local-machine utilities
    skill                Install remote-agent skill docs
    config               Manage saved server domains (local web UI)

Run 'remote-agent <command> --help' for subcommands and options.
Run 'remote-agent <command> <subcommand> --help' for nested topics.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var server string
	var token string
	tokenSpecified := hasGlobalFlag(args, "--token")

	args, err := flags.
		String("--server", &server).
		String("--token", &token).
		Help("-h,--help", help).
		StopOnFirstArg().
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		fmt.Print(help)
		return nil
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "config":
		return runConfig(rest)
	case "ping":
		return runPing(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "upload":
		if wantsHelp(rest) {
			return runUpload(nil, rest)
		}
		cli, err := resolveClient(server, token, tokenSpecified)
		if err != nil {
			return err
		}
		return runUpload(cli, rest)
	case "download":
		if wantsHelp(rest) {
			return runDownload(nil, rest)
		}
		cli, err := resolveClient(server, token, tokenSpecified)
		if err != nil {
			return err
		}
		return runDownload(cli, rest)
	case "local":
		return runLocal(rest)
	case "exec":
		return runExec(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "request":
		return runRequest(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "bash":
		return runBash(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "terminal":
		return runTerminal(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "git":
		return runGit(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "proxy":
		return runProxy(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "project":
		return runProject(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "settings":
		return runSettings(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "service":
		return runService(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "server":
		return runServer(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "auth":
		return runAuth(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "agent":
		return runAgent(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "skill":
		return remoteagentskill.Handle(rest)
	case "openclaw":
		return runOpenClaw(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	case "ws-proxy":
		return runWSProxy(func() (*client.Client, error) {
			return resolveClient(server, token, tokenSpecified)
		}, rest)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// resolveClient builds a client.Client using CLI flags, falling back to the
// default domain saved via 'remote-agent config'.
func resolveClient(server string, token string, tokenSpecified bool) (*client.Client, error) {
	cfg, _ := loadConfig()

	// When --server is not supplied, use the saved default domain's
	// server + token. --token alone can still override the saved token.
	if server == "" {
		def := cfg.DefaultDomain()
		if def == nil {
			return nil, fmt.Errorf("no server specified and no default domain configured. " +
				"Pass --server, or run 'remote-agent config' to add a domain and mark it as default.")
		}
		server = def.Server
		if token == "" {
			token = def.Token
		}
	} else if !tokenSpecified {
		if domain := cfg.FindDomain(server); domain != nil {
			token = domain.Token
		}
	}

	return client.New(server, token), nil
}

func wantsHelp(args []string) bool {
	return len(args) > 0 && (args[0] == "-h" || args[0] == "--help")
}

// hasGlobalFlag reports whether a global flag was present before the command.
// The remote-agent parser stops at the first command token, so command-specific
// arguments after that point should not affect global credential resolution.
func hasGlobalFlag(args []string, name string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return false
		}
		if arg == name || strings.HasPrefix(arg, name+"=") {
			return true
		}
		switch arg {
		case "--server", "--token":
			if i+1 < len(args) {
				i++
			}
			continue
		}
		if !strings.HasPrefix(arg, "-") {
			return false
		}
	}
	return false
}
