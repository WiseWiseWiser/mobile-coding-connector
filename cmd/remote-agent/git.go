package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

const gitHelp = `Usage: remote-agent git <subcommand> [args...]

Git utilities that run on the remote server. All subcommands stream
stdout/stderr back and mirror the remote git process's exit code.

Subcommands:
  clone [--private-key <key-file>] [--https-proxy <proxy-url>] <repo> [dir]
      Clone <repo> on the remote machine. If [dir] is omitted, the
      repository is cloned into ~/<repo_base_name>. If the target path
      already exists, the command errors out.

  -C <dir> fetch [--private-key <key-file>] [--https-proxy <proxy-url>]
      Run 'git fetch' inside <dir> on the remote machine. '-C <dir>'
      must appear right after 'git'.

  -C <dir> pull [--private-key <key-file>] [--https-proxy <proxy-url>]
      Run 'git pull --ff-only' inside <dir> on the remote machine.
      '-C <dir>' must appear right after 'git'.

Examples:
  remote-agent git clone https://github.com/foo/bar.git
  remote-agent git -C ~/bar fetch --private-key ~/.ssh/id_rsa
  remote-agent git -C ~/bar pull --private-key ~/.ssh/id_rsa
`

const gitCloneHelp = `Usage: remote-agent git clone [--private-key <key-file>] [--https-proxy <proxy-url>] <repo> [dir]

Clone <repo> on the remote machine via the server's /api/git/clone
endpoint.

Options:
  --private-key FILE   Local path to an SSH private key. The file is
                       read locally and its contents sent to the server,
                       which writes it to a temp file and sets
                       GIT_SSH_COMMAND for the clone.
  --https-proxy URL    Value the server exports as https_proxy /
                       HTTPS_PROXY for the 'git clone' process.
  -h, --help           Show this help message.

Path resolution:
  If [dir] is omitted, the server clones into ~/<repo_base_name>, where
  <repo_base_name> is the last path segment of <repo> with a trailing
  '.git' stripped. If the target already exists, the server errors out.
`

const gitFetchHelp = `Usage: remote-agent git -C <dir> fetch [--private-key <key-file>] [--https-proxy <proxy-url>]

Run 'git fetch' inside <dir> on the remote machine. <dir> must already
be a git repository on the server. '-C <dir>' must appear right after
'git' and is required for this subcommand.

Options:
  --private-key FILE   Local path to an SSH private key.
  --https-proxy URL    Value the server exports as https_proxy / HTTPS_PROXY.
  -h, --help           Show this help message.
`

const gitPullHelp = `Usage: remote-agent git -C <dir> pull [--private-key <key-file>] [--https-proxy <proxy-url>]

Run 'git pull --ff-only' inside <dir> on the remote machine. <dir> must
already be a git repository on the server. '-C <dir>' must appear right
after 'git' and is required for this subcommand.

Options:
  --private-key FILE   Local path to an SSH private key.
  --https-proxy URL    Value the server exports as https_proxy / HTTPS_PROXY.
  -h, --help           Show this help message.
`

// runGit dispatches 'remote-agent git [-C <dir>] <subcommand> [args...]'.
//
// The optional '-C <dir>' prefix must appear right after 'git'. It is
// required for fetch/pull (which operate on an existing repo) and
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
	case "-h", "--help":
		fmt.Print(gitHelp)
		return nil
	default:
		return fmt.Errorf("unknown git subcommand: %s", sub)
	}
}

// runGitClone invokes the server's /api/git/clone endpoint.
func runGitClone(resolve func() (*client.Client, error), args []string) error {
	var privateKey string
	var httpsProxy string

	args, err := flags.
		String("--private-key", &privateKey).
		String("--https-proxy", &httpsProxy).
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

// runGitRepoOp handles the fetch and pull subcommands, which share the
// same flag set and request shape.
func runGitRepoOp(resolve func() (*client.Client, error), dir string, op string, args []string) error {
	var privateKey string
	var httpsProxy string

	helpText := gitFetchHelp
	if op == "pull" {
		helpText = gitPullHelp
	}

	args, err := flags.
		String("--private-key", &privateKey).
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
		HTTPSProxy: httpsProxy,
	}
	var exitCode int
	switch op {
	case "fetch":
		exitCode, err = cli.GitFetchWithKeyFile(req, privateKey, gitStreamHandler())
	case "pull":
		exitCode, err = cli.GitPullWithKeyFile(req, privateKey, gitStreamHandler())
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
