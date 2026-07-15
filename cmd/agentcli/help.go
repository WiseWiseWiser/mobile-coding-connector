package agentcli

import "fmt"

func topLevelHelp(p Profile) string {
	if p.Name == "local-agent" {
		return fmt.Sprintf(`Usage: local-agent [--server URL] [--port PORT] [--token TOKEN] <command> [args...]

CLI for the local ai-critic server API. Configure a default server with 'config';
then most commands use saved credentials.

Global options:
  --server URL    Server base URL (default: saved domain or http://localhost:%d)
  --port PORT     Shorthand for --server http://localhost:PORT (mutually exclusive with --server)
%d           Built-in default port when no --server, --port, or saved config
  --token TOKEN   Bearer token (default: saved token for --server)
  -h, --help      Show this help

Getting started:
  local-agent config          # pick default server + token
  local-agent ping            # check reachability
  local-agent auth status     # verify token

Commands:
  Connectivity
    ping                 Check server reachability
    auth                 Authentication checks

  Files
    upload               Upload a local file to the server
    download             Download a remote file
    paste-bin            Read/write the Quick Transfer scratch pad

  Remote shell
    exec                 Run a command on the server
    bash                 Interactive remote shell
    terminal             Persistent terminal sessions

  Git & projects
    git                  Server-side git operations
    project              Project metadata and git identity
    settings             Server settings (git users, etc.)

  Machine
    machine              Backup/restore server HOME dot-files and dot-dirs

  Operations
    service              Managed services (start/stop/logs/upgrade)
    cron                 Scheduled cron tasks (interval or UTC cron)
    server               Server lifecycle (build-next, restart, status)
    proxy                Configured HTTP proxies
    request              Call arbitrary API paths

  Integrations
    agent                Custom agents and sessions
    openclaw             Mock OpenClaw gateway + Slack config
    ws-proxy             Mobile WebSocket proxy (Xray + tunnel)

  Local & tooling
    local                Local-machine utilities
    skill                Install local-agent skill docs
    config               Manage saved server domains (local web UI)

Run 'local-agent <command> --help' for subcommands and options.
Run 'local-agent <command> <subcommand> --help' for nested topics.`, p.DefaultPort, p.DefaultPort)
	}
	return `Usage: remote-agent [--server URL] [--token TOKEN] <command> [args...]

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
    paste-bin            Read/write the Quick Transfer scratch pad

  Remote shell
    exec                 Run a command on the server
    bash                 Interactive remote shell
    terminal             Persistent terminal sessions

  Git & projects
    git                  Server-side git operations
    project              Project metadata and git identity
    settings             Server settings (git users, etc.)

  Machine
    machine              Backup/restore server HOME dot-files and dot-dirs

  Operations
    service              Managed services (start/stop/logs/upgrade)
    cron                 Scheduled cron tasks (interval or UTC cron)
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
Run 'remote-agent <command> <subcommand> --help' for nested topics.`
}