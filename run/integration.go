package run

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/xhd2015/less-gen/flags"
)

const integrationHelp = `Usage: ai-critic integration <target> <command> [OPTIONS]

Host-OS service integration.

Targets:
  systemctl    systemd unit for this binary (Linux)

Run 'ai-critic integration <target> --help' for target-specific commands.
`

const integrationSystemctlHelp = `Usage: ai-critic integration systemctl <command> [OPTIONS]

Install and control a systemd unit that runs this binary (not keep-alive).

Commands:
  status       systemctl status + /ping
  start        ensure unit file, enable, start (restart if already active)
  stop         systemctl stop
  logs         journalctl -u ai-critic-server (default -n 100 -f)
  show-config  generated unit, on-disk diff, day-to-day commands

Options:
  --port N     server port baked into ExecStart (default 23712)
  --lines N    logs: journal lines (default 100)
  --no-follow  logs: print and exit (no -f)
  --dry-run    print unit + systemctl/journalctl actions, do not change the host
  -h, --help
`

func runIntegration(args []string) error {
	if len(args) == 0 || isRunHelpToken(args[0]) {
		fmt.Fprint(integrationStdout(), strings.TrimPrefix(integrationHelp, "\n"))
		return nil
	}
	switch args[0] {
	case "systemctl":
		return runIntegrationSystemctl(args[1:])
	default:
		return fmt.Errorf("unknown integration target %q (want systemctl)", args[0])
	}
}

func isRunHelpToken(s string) bool {
	return s == "-h" || s == "--help" || s == "help"
}

func integrationStdout() io.Writer {
	if integrationOut != nil {
		return integrationOut
	}
	return os.Stdout
}

func integrationStderr() io.Writer {
	if integrationErr != nil {
		return integrationErr
	}
	return os.Stderr
}

type sysctlFlags struct {
	DryRun   bool
	Port     int
	Lines    int
	NoFollow bool
}

func parseSystemctlFlags(args []string, help string) (sysctlFlags, []string, error) {
	var f sysctlFlags
	rest, err := flags.
		Bool("--dry-run", &f.DryRun).
		Int("--port", &f.Port).
		Int("--lines", &f.Lines).
		Bool("--no-follow", &f.NoFollow).
		Help("-h,--help", help).
		HelpNoExit().
		StopOnFirstArg().
		Parse(args)
	if err != nil {
		return f, nil, err
	}
	return f, rest, nil
}
