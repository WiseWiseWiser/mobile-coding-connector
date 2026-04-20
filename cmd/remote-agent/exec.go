package main

import (
	"fmt"
	"os"

	"github.com/xhd2015/lifelog-private/ai-critic/client"
)

const execHelp = `Usage: remote-agent exec <BINARY> [ARGS...]

Run a subprocess on the server. Stdout and stderr are streamed back to this
machine as they are produced; the client's exit code mirrors the remote
exit code.

Every argument after 'exec' is forwarded verbatim to the remote process,
so there is no need for '--' or client-side flag parsing.

Examples:
  remote-agent exec ls -la /tmp
  remote-agent exec sh -c 'echo hi; sleep 1'
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
