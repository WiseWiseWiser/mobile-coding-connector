package agentcli

import (
	"fmt"
	"os"

	"golang.org/x/term"

	ptyclient "github.com/xhd2015/dot-pkgs/go-pkgs/shell/ptywrap/client"
	"github.com/xhd2015/ai-critic/client"
)

const execHelp = `Usage: remote-agent exec <BINARY> [ARGS...]

Run a subprocess on the server. In non-interactive mode, stdout and stderr
are streamed back to this machine as they are produced; the client's exit
code mirrors the remote exit code.

When stdin/stdout are attached to an interactive terminal, 'exec' switches
to a PTY-backed mode so the remote process can receive live user input.

Every argument after 'exec' is forwarded verbatim to the remote process,
so there is no need for '--' or client-side flag parsing.

Examples:
  remote-agent exec ls -la /tmp
  remote-agent exec sh -c 'echo hi; sleep 1'
  remote-agent exec python3
`

// runExec is the client-side implementation of 'remote-agent exec'.
//
// By design, this subcommand does NOT use a flag parser: every argument
// after 'exec' is passed verbatim to the remote binary, so users can invoke
// commands with flags of their own (e.g. 'remote-agent exec ls -la') without
// needing '--'. The only recognized client-side token is '--help' / '-h' as
// the first argument, matching common CLI conventions.
func runExec(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("exec requires <BINARY> [ARGS...]; see 'remote-agent exec --help'")
	}
	if args[0] == "-h" || args[0] == "--help" {
		fmt.Print(execHelp)
		return nil
	}
	if term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd())) {
		return runExecInteractive(resolve, args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	exitCode, err := cli.Exec(client.ExecRequest{Argv: args}, func(ev client.ExecEvent) {
		switch ev.Type {
		case "stdout":
			os.Stdout.WriteString(ev.Data)
		case "stderr":
			os.Stderr.WriteString(ev.Data)
		}
	})
	if err != nil {
		return err
	}

	if exitCode != 0 {
		// Exit with the same code the remote process produced, so
		// scripts wrapping 'remote-agent exec' behave correctly.
		os.Exit(normalizeExitCode(exitCode))
	}
	return nil
}

func runExecInteractive(resolve func() (*client.Client, error), args []string) error {
	cli, err := resolve()
	if err != nil {
		return err
	}

	c := &ptyclient.Client{
		BaseURL:   cli.Server,
		AuthToken: cli.Token,
	}
	exitCode, err := ptyclient.RunExec(c, ptyclient.ExecOptions{Argv: args})
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(normalizeExitCode(exitCode))
	}
	return nil
}

// normalizeExitCode clamps an arbitrary integer into the 1..255 range that
// os.Exit accepts portably. -1 (our "killed by signal" sentinel) becomes 255.
func normalizeExitCode(code int) int {
	if code <= 0 {
		return 255
	}
	if code > 255 {
		return 255
	}
	return code
}