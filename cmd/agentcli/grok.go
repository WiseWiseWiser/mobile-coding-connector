package agentcli

import (
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	ptyclient "github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap/client"
	"github.com/xhd2015/less-gen/flags"
	"golang.org/x/term"
)

const grokHelp = `Usage: remote-agent grok [--cwd DIR] [--resume SESSION_ID]

Start or resume a Grok session on the remote server and attach this terminal
tab to its PTY (no new window/tab).

  remote-agent grok
      Run remote "grok" (new session) and attach.

  remote-agent grok --resume SESSION_ID
      Run remote "grok --resume SESSION_ID" and attach.
      SESSION_ID is a Grok runner session id (e.g. 01a09d34-…).

Options:
  --cwd DIR           Working directory on the remote server.
                      For --resume, defaults to the workspace encoded under
                      ~/.grok/sessions/<urlencoded-cwd>/<SESSION_ID>/ when found.
  --resume SESSION_ID Resume an existing Grok session id.
  -h, --help          Show this help.

Detach without stopping remote Grok: Ctrl-].
Re-attach later: remote-agent terminal attach <id-or-name>
`

func runGrok(resolve func() (*client.Client, error), args []string) error {
	var cwd, resumeID string
	args, err := flags.
		String("--cwd", &cwd).
		String("--resume", &resumeID).
		Help("-h,--help", grokHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("grok takes no positional arguments; use --resume SESSION_ID (got %v)", args)
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("remote-agent grok requires an interactive terminal on stdin/stdout")
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	resumeID = strings.TrimSpace(resumeID)
	cwd = strings.TrimSpace(cwd)

	if resumeID != "" && cwd == "" {
		ws, err := cli.ResolveGrokSessionWorkspace(resumeID)
		if err != nil {
			return err
		}
		if ws != "" {
			cwd = ws
			fmt.Fprintf(os.Stderr, "notice: resume cwd %s (from grok session store)\n", cwd)
		}
	}
	if cwd == "" {
		home, err := cli.GetHome()
		if err != nil {
			return err
		}
		cwd = home.Home
	}

	command := []string{"grok"}
	name := "grok"
	if resumeID != "" {
		command = append(command, "--resume", resumeID)
		short := resumeID
		if len(short) > 8 {
			short = short[:8]
		}
		name = "grok-" + short
	}

	pty := ptyClientFrom(cli)
	info, err := pty.Create(command, cwd, name)
	if err != nil {
		return fmt.Errorf("start remote grok: %w", err)
	}
	fmt.Fprintf(os.Stderr, "notice: started %s (pty %s) in %s\n", name, info.ID, cwd)
	if resumeID != "" {
		fmt.Fprintf(os.Stderr, "notice: grok --resume %s\n", resumeID)
	}

	result, err := ptyclient.Attach(pty, TerminalAttachConnectOptions(info.ID))
	if err != nil {
		return err
	}
	printDetached(result)
	return nil
}
