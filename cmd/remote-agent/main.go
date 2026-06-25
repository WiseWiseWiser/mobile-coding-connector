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

  ping
      Ping the server (GET /ping, expects "pong") and report reachability.

  upload <LOCAL_FILE> [REMOTE_PATH]
      Upload a local file to the server using chunked upload.
      If REMOTE_PATH is omitted, the file's basename is used.
      If REMOTE_PATH ends with '/', the basename is appended.

  download <REMOTE_PATH> [LOCAL_PATH]
      Download a remote file from the server.
      REMOTE_PATH may use ~/ to refer to the server's home directory.
      If LOCAL_PATH is omitted, the remote file's basename is used.

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

  request <api-path> [json-body]
      Call an arbitrary API endpoint on the remote server. Uses GET when
      json-body is omitted, POST with application/json when supplied or
      piped on stdin.

  bash [cwd]
      Start an interactive shell on the remote server using the same
      terminal WebSocket API as the frontend terminal page. The server-side
      terminal stays alive after the client disconnects.

  terminal <subcommand> [args...]
      Manage persistent remote terminal sessions. Subcommands:
        list
            List terminal sessions, including exited ones.
        new [--name NAME]
            Create a detached terminal session on the server.
        close <id-or-name>
            Remove a terminal session from the server.
        attach <id-or-name>
            Attach this terminal to an existing remote terminal session.

  git <subcommand> [args...]
      Git utilities that run on the remote server. Subcommands:
        clone [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>] <repo-or-remote-dir> [dir]
            Clone <repo-or-remote-dir> on the remote machine. If [dir] is omitted, the
            repository is cloned into ~/<repo_base_name>. If the target
            already exists, the command errors out.
        -C <dir> fetch [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>]
            Run 'git fetch' inside <dir> on the remote machine.
        -C <dir> pull [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>]
            Run 'git pull --ff-only' inside <dir> on the remote machine.
        -C <dir> push [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>]
            Run 'git push origin HEAD:<current-branch>' inside <dir> on
            the remote machine.

  proxy <subcommand> [args...]
      Inspect proxy servers configured in the remote server's settings.
      Subcommands:
        list
            List all configured proxy servers. Passwords are masked.

  project <subcommand> [args...]
      Inspect and update projects known to the remote server. Subcommands:
        list
            List projects and their configured Git commit identity.
        git-config get|check <project-id-or-name-or-dir>
            Show the Git commit identity configured for one project.
        git-config set <project-id-or-name-or-dir> --name NAME --email EMAIL [--identity-id ID]
            Set the Git commit identity used by this project.
        git-config unset <project-id-or-name-or-dir>
            Clear the Git commit identity for this project.

  settings <subcommand> [args...]
      Manage remote server settings. Subcommands:
        git-users list
            List configured Git commit identities.
        git-users add --name NAME --email EMAIL [--id ID]
            Add a Git commit identity.
        git-users set <id> --name NAME --email EMAIL
            Update a Git commit identity.
        git-users delete <id>
            Delete a Git commit identity.

  service <subcommand> [args...]
      Manage services exposed by the frontend's Services tab.
      Subcommands:
        list
            List all managed services.
        stop|start|restart|logs <service-name-or-id>
            Control one service or stream its logs.
        rename <service-name-or-id> <new-name>
            Rename one service without restarting it.
        update <service-name-or-id> [--field value...]
            Update one service definition without restarting it.
        upgrade <service-name-or-id> <local-binary> [--target <remote-path>]
            Upload a binary first, then stop, replace, and start the service.

  server <subcommand> [args...]
      Execute server-management actions exposed by the remote server UI.
      Subcommands:
        build-next [--project <id>]
            Trigger the same "Build Next" action as the Manage Server page.
        upload-next <local-binary>
            Upload a local binary to the next remote ai-critic-server-vN path.
        restart
            Trigger the same "Restart Server" action as the Manage Server page.
        status
            Show the same keep-alive and machine status as the Manage Server page.

   auth <subcommand> [args...]
       Check authentication status against the configured server.
       Subcommands:
         status
             Verify the server is reachable and the token is valid.

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
         show
             Print the content of SKILL.md.
         install [<dir>] [--cursor|--codex]
             Install the packaged SKILL.md for Cursor or Codex.

   ws-proxy <subcommand> [args...]
       Manage the WebSocket-based mobile proxy (Xray + Cloudflare Tunnel).
       Subcommands:
         start [--tmp] [--upstream-proxy URL]
             Start the proxy. With --tmp, uses temporary Quick Tunnel.
         stop
             Stop the proxy.
         status
             Show proxy status.
         config
             Show current configuration.
         config set --upstream-proxy URL [--port PORT] [--path PATH]
             Update configuration.
         vmess-link [--export FILE]
             Get the vmess:// link, manual config, and QR code for Shadowrocket import.
         doctor [--try-url URL]
             Diagnose ws-proxy health (server + client). Default tests google.com.

Examples:
  remote-agent config
  remote-agent ping
  remote-agent --server https://host.example.com --token abc upload ./foo.txt /tmp/foo.txt
  remote-agent upload ./foo.txt /tmp/          # uses saved config
  remote-agent download '~/server.log'
  remote-agent download /tmp/foo.txt ./foo.txt
  remote-agent local reap                       # list zombies
  remote-agent local reap --filter ai-critic --signal
  remote-agent exec ls -la /tmp
  remote-agent exec sh -c 'echo hi; sleep 1'
  remote-agent request /api/services
  remote-agent request /api/services/start?id=svc-123 '{}'
  remote-agent bash
  remote-agent bash ~/work/repo
  remote-agent terminal list
  remote-agent terminal new --name Debug
  remote-agent terminal attach session-1
  remote-agent terminal close Debug
  remote-agent git clone https://github.com/foo/bar.git
  remote-agent git clone --private-key ~/.ssh/id_rsa git@host:foo/bar.git /tmp/bar
  remote-agent git clone https://github.com/foo/private-bar.git ~/bar --git-token ghp_example
  remote-agent git -C ~/bar fetch --private-key ~/.ssh/id_rsa
  remote-agent git -C ~/bar pull --private-key ~/.ssh/id_rsa
  remote-agent git -C ~/bar push --private-key ~/.ssh/id_rsa
  remote-agent proxy list
  remote-agent project list
  remote-agent project git-config check my-project
  remote-agent project git-config set my-project --name "Jane Doe" --email jane@example.com
  remote-agent settings git-users list
  remote-agent settings git-users add --name "Jane Doe" --email jane@example.com
  remote-agent service list
  remote-agent service restart web
  remote-agent service rename web api
  remote-agent service update api --command './server --port 8080'
  remote-agent service upgrade web ./ai-critic-server-linux-amd64
  remote-agent service logs svc-123
  remote-agent server build-next
  remote-agent server upload-next ./ai-critic-server-linux-amd64
  remote-agent server restart
   remote-agent server status
   remote-agent auth status
   remote-agent agent list
  remote-agent agent add build-review --template build --name "Build Review"
  remote-agent agent sessions build-review
  remote-agent agent run build-review --project ~/work/repo
  remote-agent skill show
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
		cli, err := resolveClient(server, token, tokenSpecified)
		if err != nil {
			return err
		}
		return runUpload(cli, rest)
	case "download":
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
