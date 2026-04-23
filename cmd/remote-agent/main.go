package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/client"
	remoteagentskill "github.com/xhd2015/lifelog-private/ai-critic/cmd/remote-agent/skill"
)

const help = `Usage: remote-agent [--server URL] [--token TOKEN] <command> [args...]

A CLI that talks to the ai-critic server's HTTP API, mirroring the
behavior of the web frontend.

Global options:
  --server URL       Base URL of the server (e.g. https://host.example.com).
                     Falls back to the default domain saved by 'config'.
  --token TOKEN      Bearer token for authentication. Falls back to the
                     token of the matching/default saved domain.
  -h, --help         Show this help message

Commands:
  config
      Open a local web page to manage saved server domains and pick a
      default one. When 'upload' is invoked without --server, the
      default domain's server and token are used.

  upload <LOCAL_FILE> [REMOTE_PATH]
      Upload a local file to the server using chunked upload.
      If REMOTE_PATH is omitted, the file's basename is used.
      If REMOTE_PATH ends with '/', the basename is appended.

  local <subcommand> [args...]
      Local-machine utilities. Subcommands:
        reap [--signal] [--kill-parent] [--filter NAME]
            List defunct (zombie) processes on the local machine. With
            --signal, send SIGCHLD to their parents to nudge reaping.
            With --kill-parent, SIGTERM the parents so init adopts and
            reaps them.

  exec <BINARY> [ARGS...]
      Run a subprocess on the server. Stdout and stderr are streamed back
      as they are produced; the client's exit code mirrors the remote exit
      code. When launched from an interactive terminal, stdin is forwarded
      through a PTY so commands can prompt for user input. Every argument
      after 'exec' is forwarded verbatim to the remote binary.

  bash [cwd]
      Start an interactive shell on the remote server using the same
      terminal WebSocket API as the frontend terminal page.

  git <subcommand> [args...]
      Git utilities that run on the remote server. Subcommands:
        clone [--private-key <key-file>] [--https-proxy <proxy-url>] <repo> [dir]
            Clone <repo> on the remote machine. If [dir] is omitted, the
            repository is cloned into ~/<repo_base_name>. If the target
            already exists, the command errors out.
        -C <dir> fetch [--private-key <key-file>] [--https-proxy <proxy-url>]
            Run 'git fetch' inside <dir> on the remote machine.
        -C <dir> pull [--private-key <key-file>] [--https-proxy <proxy-url>]
            Run 'git pull --ff-only' inside <dir> on the remote machine.
        -C <dir> push [--private-key <key-file>] [--https-proxy <proxy-url>]
            Run 'git push origin HEAD:<current-branch>' inside <dir> on
            the remote machine.

  proxy <subcommand> [args...]
      Inspect proxy servers configured in the remote server's settings.
      Subcommands:
        list
            List all configured proxy servers. Passwords are masked.

  server <subcommand> [args...]
      Execute server-management actions exposed by the remote server UI.
      Subcommands:
        build-next [--project <id>]
            Trigger the same "Build Next" action as the Manage Server page.
        restart
            Trigger the same "Restart Server" action as the Manage Server page.
        status
            Show the same keep-alive and machine status as the Manage Server page.

  agent <subcommand> [args...]
      Manage remote custom agents and their saved sessions. Subcommands:
        list
            List custom agents.
        show <agent-id>
            Show one custom agent as JSON.
        add [<agent-id>] [options...]
            Create a custom agent.
        delete <agent-id>
            Delete a custom agent.
        sessions <agent-id>
            List saved sessions for one custom agent.
        run <agent-id> [--project <dir>] [--resume <session-id|latest>]
            Start a new session or resume an existing saved session.

  skill <subcommand> [args...]
      Manage the embedded remote-agent skill definition. Subcommands:
        install [<dir>] [--cursor|--codex]
            Install the packaged SKILL.md for Cursor or Codex.

Examples:
  remote-agent config
  remote-agent --server https://host.example.com --token abc upload ./foo.txt /tmp/foo.txt
  remote-agent upload ./foo.txt /tmp/          # uses saved config
  remote-agent local reap                       # list zombies
  remote-agent local reap --filter ai-critic --signal
  remote-agent exec ls -la /tmp
  remote-agent exec sh -c 'echo hi; sleep 1'
  remote-agent bash
  remote-agent bash ~/work/repo
  remote-agent git clone https://github.com/foo/bar.git
  remote-agent git clone --private-key ~/.ssh/id_rsa git@host:foo/bar.git /tmp/bar
  remote-agent git -C ~/bar fetch --private-key ~/.ssh/id_rsa
  remote-agent git -C ~/bar pull --private-key ~/.ssh/id_rsa
  remote-agent git -C ~/bar push --private-key ~/.ssh/id_rsa
  remote-agent proxy list
  remote-agent server build-next
  remote-agent server restart
  remote-agent server status
  remote-agent agent list
  remote-agent agent add build-review --template build --name "Build Review"
  remote-agent agent sessions build-review
  remote-agent agent run build-review --project ~/work/repo
  remote-agent skill install --codex
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
	case "upload":
		cli, err := resolveClient(server, token)
		if err != nil {
			return err
		}
		return runUpload(cli, rest)
	case "local":
		return runLocal(rest)
	case "exec":
		return runExec(func() (*client.Client, error) {
			return resolveClient(server, token)
		}, rest)
	case "bash":
		return runBash(func() (*client.Client, error) {
			return resolveClient(server, token)
		}, rest)
	case "git":
		return runGit(func() (*client.Client, error) {
			return resolveClient(server, token)
		}, rest)
	case "proxy":
		return runProxy(func() (*client.Client, error) {
			return resolveClient(server, token)
		}, rest)
	case "server":
		return runServer(func() (*client.Client, error) {
			return resolveClient(server, token)
		}, rest)
	case "agent":
		return runAgent(func() (*client.Client, error) {
			return resolveClient(server, token)
		}, rest)
	case "skill":
		return remoteagentskill.Handle(rest)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// resolveClient builds a client.Client using CLI flags, falling back to the
// default domain saved via 'remote-agent config'.
func resolveClient(server string, token string) (*client.Client, error) {
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
	}

	return client.New(server, token), nil
}
