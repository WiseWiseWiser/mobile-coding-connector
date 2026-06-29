package agentcli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const gitHelp = `Usage: remote-agent git <subcommand> [args...]

Git utilities that run on the remote server. All subcommands stream
stdout/stderr back and mirror the remote git process's exit code.

Subcommands:
  clone [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>] <repo-or-remote-dir> [dir]
      Clone <repo-or-remote-dir> on the remote machine. If [dir] is omitted, the
      repository is cloned into ~/<repo_base_name>. If the target path
      already exists, the command errors out.

  -C <dir> fetch [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>]
      Run 'git fetch' inside <dir> on the remote machine. '-C <dir>'
      must appear right after 'git'.

  -C <dir> pull [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>]
      Run 'git pull --ff-only' inside <dir> on the remote machine.
      '-C <dir>' must appear right after 'git'.

  -C <dir> push [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>]
      Run 'git push origin HEAD:<current-branch>' inside <dir> on the
      remote machine. '-C <dir>' must appear right after 'git'.

  -C <dir> status|diff|log|branch|rev-parse|show|remote|config|stash
      [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>]
      [git-args...]
      Run an allowlisted read-only git subcommand inside <dir> on the remote
      machine. '-C <dir>' must appear right after 'git'.

Examples:
  remote-agent git clone https://github.com/foo/bar.git
  remote-agent git clone ~/src/bar ~/bar --git-token ghp_example
  remote-agent git -C ~/bar fetch --private-key ~/.ssh/id_rsa
  remote-agent git -C ~/bar pull --private-key ~/.ssh/id_rsa
  remote-agent git -C ~/bar push --private-key ~/.ssh/id_rsa
  remote-agent git -C ~/bar status
  remote-agent git -C ~/bar diff --cached
  remote-agent git -C ~/bar log --oneline -2
  remote-agent git -C ~/bar stash list
`

const gitCloneHelp = `Usage: remote-agent git clone [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>] [--ssh-user <user>] <repo-or-remote-dir> [dir]

Clone <repo-or-remote-dir> on the remote machine via the server's
/api/remote-agent/git/clone endpoint.

Options:
  --private-key FILE   Local path to an SSH private key. The file is
                       read locally and its contents sent to the server,
                       which writes it to a temp file and sets
                       GIT_SSH_COMMAND for the clone.
  --git-token TOKEN    HTTPS git auth token. The server exposes it to
                       git through a temporary GIT_ASKPASS helper.
  --https-proxy URL    Value the server exports as https_proxy /
                       HTTPS_PROXY for HTTPS git traffic, and also uses
                       as the SSH proxy when the repo is rewritten to
                       SSH.
  --ssh-user USER      SSH user to use when the server rewrites an HTTPS
                       <repo> to SSH. Only consulted together with
                       --private-key. Defaults to 'git'. Use 'gitlab'
                       for self-hosted GitLab instances that require
                       it (e.g. git.garena.com).
  -h, --help           Show this help message.

URL rewriting:
  If <repo> is an HTTPS URL and --private-key is supplied, the server
  rewrites it to the scp-like SSH form (<ssh-user>@host:path) before
  cloning, so the provided key is actually used. Pass an SSH URL
  directly to skip this rewrite.

Path resolution:
  <repo-or-remote-dir> may be a git URL or a path on the remote machine.
  Remote paths using '~' are expanded on the server. If [dir] is omitted,
  the server clones into ~/<repo_base_name>, where <repo_base_name> is
  the last path segment with a trailing '.git' stripped. If the target
  already exists, the server errors out.
`

const gitFetchHelp = `Usage: remote-agent git -C <dir> fetch [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>]

Run 'git fetch' inside <dir> on the remote machine. <dir> must already
be a git repository on the server. '-C <dir>' must appear right after
'git' and is required for this subcommand.

Options:
  --private-key FILE   Local path to an SSH private key.
  --git-token TOKEN    HTTPS git auth token.
  --https-proxy URL    Value the server exports as https_proxy / HTTPS_PROXY.
  -h, --help           Show this help message.
`

const gitPullHelp = `Usage: remote-agent git -C <dir> pull [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>]

Run 'git pull --ff-only' inside <dir> on the remote machine. <dir> must
already be a git repository on the server. '-C <dir>' must appear right
after 'git' and is required for this subcommand.

Options:
  --private-key FILE   Local path to an SSH private key.
  --git-token TOKEN    HTTPS git auth token.
  --https-proxy URL    Value the server exports as https_proxy / HTTPS_PROXY.
  -h, --help           Show this help message.
`

const gitLocalHelp = `Usage: remote-agent git -C <dir> <subcommand> [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>] [git-args...]

Run an allowlisted read-only git subcommand inside <dir> on the remote
machine. <subcommand> is one of: status, diff, log, branch, rev-parse,
show, remote, config, stash. '-C <dir>' must appear right after 'git'
and is required.

Options:
  --private-key FILE   Local path to an SSH private key.
  --git-token TOKEN    HTTPS git auth token.
  --https-proxy URL    Value the server exports as https_proxy / HTTPS_PROXY.
  -h, --help           Show this help message.
`

const gitPushHelp = `Usage: remote-agent git -C <dir> push [--private-key <key-file>] [--git-token <token>] [--https-proxy <proxy-url>]

Run 'git push origin HEAD:<current-branch>' inside <dir> on the remote
machine. <dir> must already be a git repository on the server. '-C <dir>'
must appear right after 'git' and is required for this subcommand.

Options:
  --private-key FILE   Local path to an SSH private key.
  --git-token TOKEN    HTTPS git auth token.
  --https-proxy URL    Value the server exports as https_proxy / HTTPS_PROXY.
  -h, --help           Show this help message.
`

var gitLocalSubcommands = map[string]struct{}{
	"status":    {},
	"diff":      {},
	"log":       {},
	"branch":    {},
	"rev-parse": {},
	"show":      {},
	"remote":    {},
	"config":    {},
	"stash":     {},
}

// runGit dispatches 'remote-agent git [-C <dir>] <subcommand> [args...]'.
//
// The optional '-C <dir>' prefix must appear right after 'git'. It is
// required for fetch/pull/push (which operate on an existing repo) and
// rejected for clone (which has its own positional <dir>).
func runGit(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(gitHelp)
		return nil
	}

	var cDir string
	var cDirSet bool

	// Consume an optional '-C <dir>' prefix. We handle it manually
	// rather than via the flags package so the subcommand can own its
	// own flag parser without collisions.
	if args[0] == "-C" {
		if len(args) < 2 {
			return fmt.Errorf("'-C' requires a directory argument")
		}
		cDir = args[1]
		cDirSet = true
		args = args[2:]
	}

	if len(args) == 0 {
		return fmt.Errorf("git requires a subcommand; see 'remote-agent git --help'")
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "clone":
		if cDirSet {
			return fmt.Errorf("'-C <dir>' is not supported for 'git clone'; pass <dir> as a positional argument instead")
		}
		return runGitClone(resolve, rest)
	case "fetch":
		if !cDirSet {
			return fmt.Errorf("'git fetch' requires '-C <dir>' between 'git' and 'fetch'")
		}
		return runGitRepoOp(resolve, cDir, "fetch", rest)
	case "pull":
		if !cDirSet {
			return fmt.Errorf("'git pull' requires '-C <dir>' between 'git' and 'pull'")
		}
		return runGitRepoOp(resolve, cDir, "pull", rest)
	case "push":
		if !cDirSet {
			return fmt.Errorf("'git push' requires '-C <dir>' between 'git' and 'push'")
		}
		return runGitRepoOp(resolve, cDir, "push", rest)
	case "-h", "--help":
		fmt.Print(gitHelp)
		return nil
	default:
		if _, ok := gitLocalSubcommands[sub]; ok {
			if !cDirSet {
				return fmt.Errorf("'git %s' requires '-C <dir>' between 'git' and '%s'", sub, sub)
			}
			return runGitLocal(resolve, cDir, sub, rest)
		}
		return fmt.Errorf("unknown git subcommand: %s", sub)
	}
}

// runGitClone invokes the server's /api/remote-agent/git/clone endpoint.
func runGitClone(resolve func() (*client.Client, error), args []string) error {
	var privateKey string
	var gitToken string
	var httpsProxy string
	var sshUser string

	args, err := flags.
		String("--private-key", &privateKey).
		String("--git-token", &gitToken).
		String("--https-proxy", &httpsProxy).
		String("--ssh-user", &sshUser).
		Help("-h,--help", gitCloneHelp).
		Parse(args)
	if err != nil {
		return err
	}

	if len(args) < 1 {
		return fmt.Errorf("git clone requires <repo> [dir]; see 'remote-agent git clone --help'")
	}
	if len(args) > 2 {
		return fmt.Errorf("git clone takes at most 2 positional arguments, got %d", len(args))
	}

	repo := args[0]
	var dir string
	if len(args) == 2 {
		dir = args[1]
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	exitCode, err := cli.GitCloneWithKeyFile(client.GitCloneRequest{
		Repo:       repo,
		Dir:        dir,
		Token:      gitToken,
		HTTPSProxy: httpsProxy,
		SSHUser:    sshUser,
	}, privateKey, gitStreamHandler())
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(normalizeExitCode(exitCode))
	}
	return nil
}

// parseGitLocalFlags extracts remote-agent auth flags from git passthrough
// args, leaving all other tokens (including git's own flags) untouched.
func parseGitLocalFlags(args []string) (privateKey, gitToken, httpsProxy string, rest []string, err error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h", arg == "--help":
			fmt.Print(gitLocalHelp)
			return "", "", "", nil, flags.ErrHelp
		case arg == "--private-key":
			if i+1 >= len(args) {
				return "", "", "", nil, fmt.Errorf("--private-key requires a value")
			}
			i++
			privateKey = args[i]
		case arg == "--git-token":
			if i+1 >= len(args) {
				return "", "", "", nil, fmt.Errorf("--git-token requires a value")
			}
			i++
			gitToken = args[i]
		case arg == "--https-proxy":
			if i+1 >= len(args) {
				return "", "", "", nil, fmt.Errorf("--https-proxy requires a value")
			}
			i++
			httpsProxy = args[i]
		case strings.HasPrefix(arg, "--private-key="):
			privateKey = strings.TrimPrefix(arg, "--private-key=")
		case strings.HasPrefix(arg, "--git-token="):
			gitToken = strings.TrimPrefix(arg, "--git-token=")
		case strings.HasPrefix(arg, "--https-proxy="):
			httpsProxy = strings.TrimPrefix(arg, "--https-proxy=")
		default:
			rest = append(rest, arg)
		}
	}
	return privateKey, gitToken, httpsProxy, rest, nil
}

// runGitLocal handles allowlisted read-only git subcommands via
// POST /api/remote-agent/git/run.
func runGitLocal(resolve func() (*client.Client, error), dir string, sub string, args []string) error {
	privateKey, gitToken, httpsProxy, gitRest, err := parseGitLocalFlags(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}

	gitArgs := append([]string{sub}, gitRest...)

	cli, err := resolve()
	if err != nil {
		return err
	}

	exitCode, err := cli.GitRunWithKeyFile(client.GitRunRequest{
		Dir:        dir,
		Args:       gitArgs,
		Token:      gitToken,
		HTTPSProxy: httpsProxy,
	}, privateKey, gitStreamHandler())
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(normalizeExitCode(exitCode))
	}
	return nil
}

// runGitRepoOp handles the fetch, pull, and push subcommands, which share the
// same flag set and request shape.
func runGitRepoOp(resolve func() (*client.Client, error), dir string, op string, args []string) error {
	var privateKey string
	var gitToken string
	var httpsProxy string

	helpText := gitFetchHelp
	if op == "pull" {
		helpText = gitPullHelp
	} else if op == "push" {
		helpText = gitPushHelp
	}

	args, err := flags.
		String("--private-key", &privateKey).
		String("--git-token", &gitToken).
		String("--https-proxy", &httpsProxy).
		Help("-h,--help", helpText).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("git %s takes no positional arguments, got %v", op, args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	req := client.GitRepoOpRequest{
		Dir:        dir,
		Token:      gitToken,
		HTTPSProxy: httpsProxy,
	}
	var exitCode int
	switch op {
	case "fetch":
		exitCode, err = cli.GitFetchWithKeyFile(req, privateKey, gitStreamHandler())
	case "pull":
		exitCode, err = cli.GitPullWithKeyFile(req, privateKey, gitStreamHandler())
	case "push":
		exitCode, err = cli.GitPushWithKeyFile(req, privateKey, gitStreamHandler())
	default:
		return fmt.Errorf("internal: unknown git op %q", op)
	}
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(normalizeExitCode(exitCode))
	}
	return nil
}

// gitStreamHandler returns an ExecHandler that forwards stdout/stderr
// chunks to the local stdout/stderr, matching the behaviour of
// 'remote-agent exec'.
func gitStreamHandler() client.ExecHandler {
	return func(ev client.ExecEvent) {
		switch ev.Type {
		case "stdout":
			os.Stdout.WriteString(ev.Data)
		case "stderr":
			os.Stderr.WriteString(ev.Data)
		}
	}
}
