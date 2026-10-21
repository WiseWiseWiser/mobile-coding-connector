package agentcli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

func hostmonHelp() string {
	name := active.Name
	return fmt.Sprintf(`Usage: %s hostmon <subcommand>

Sample host CPU and disk I/O on the server. Off until you start or enable it.
Samples are fsynced so they survive a hard reboot.

Subcommands:
  status              Sampler state, auto-start, log path
  start               Start sampling now
  stop                Stop sampling; keep the log
  enable              Auto-start whenever ai-critic starts
  disable             Do not auto-start (default)
  top [--lines N]     Latest sample: load, iowait, disk, top pids
  log [--lines N]     Persisted samples, oldest first in the window

Run '%s hostmon top --help' or '%s hostmon log --help' for options.
`, name, name, name)
}

func hostmonTopHelp() string {
	return fmt.Sprintf(`Usage: %s hostmon top [--lines N]

Show the latest persisted sample. When the sampler is stopped, the last
sample is still printed.

Options:
  --lines N           How many latest samples to show. Defaults to 1.
  -h, --help          Show this help message.
`, active.Name)
}

func hostmonLogHelp() string {
	return fmt.Sprintf(`Usage: %s hostmon log [--lines N]

Print persisted samples, oldest first within the window.

Options:
  --lines N           How many lines to show. Defaults to 20.
  -h, --help          Show this help message.
`, active.Name)
}

func runHostmon(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 || isHelpToken(args[0]) {
		fmt.Fprint(osStdout(), hostmonHelp())
		return nil
	}
	switch args[0] {
	case "status":
		return runHostmonStatus(resolve, args[1:])
	case "start", "stop", "enable", "disable":
		return runHostmonControl(resolve, args[0], args[1:])
	case "top":
		return runHostmonTop(resolve, args[1:])
	case "log":
		return runHostmonLog(resolve, args[1:])
	default:
		return fmt.Errorf("unknown hostmon subcommand: %s", args[0])
	}
}

func runHostmonStatus(resolve func() (*client.Client, error), args []string) error {
	args, err := flags.Help("-h,--help", hostmonHelp()).Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("hostmon status takes no arguments")
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	status, err := cli.HostMonStatus()
	if err != nil {
		return err
	}
	fmt.Fprint(osStdout(), formatHostMonStatus(status))
	return nil
}

func runHostmonControl(resolve func() (*client.Client, error), action string, args []string) error {
	if len(args) > 0 && isHelpToken(args[0]) {
		fmt.Fprintf(osStdout(), "Usage: %s hostmon %s\n", active.Name, action)
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("hostmon %s takes no arguments", action)
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	if _, err := cli.HostMonControl(action); err != nil {
		return err
	}
	switch action {
	case "start":
		fmt.Fprintln(osStdout(), "hostmon started")
	case "stop":
		fmt.Fprintln(osStdout(), "hostmon stopped")
	case "enable":
		fmt.Fprintln(osStdout(), "hostmon auto-start on")
	case "disable":
		fmt.Fprintln(osStdout(), "hostmon auto-start off")
	}
	return nil
}

func runHostmonTop(resolve func() (*client.Client, error), args []string) error {
	lines := 1
	args, err := flags.Int("--lines", &lines).Help("-h,--help", hostmonTopHelp()).Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("hostmon top takes no positional arguments")
	}
	if lines < 1 {
		return fmt.Errorf("lines must be a positive integer")
	}
	return printHostMonSamples(resolve, lines, true)
}

func runHostmonLog(resolve func() (*client.Client, error), args []string) error {
	lines := 20
	args, err := flags.Int("--lines", &lines).Help("-h,--help", hostmonLogHelp()).Parse(args)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("hostmon log takes no positional arguments")
	}
	if lines < 1 {
		return fmt.Errorf("lines must be a positive integer")
	}
	return printHostMonSamples(resolve, lines, false)
}

func printHostMonSamples(resolve func() (*client.Client, error), lines int, asTop bool) error {
	cli, err := resolve()
	if err != nil {
		return err
	}
	status, err := cli.HostMonStatus()
	if err != nil {
		return err
	}
	samples, err := cli.HostMonSamples(lines)
	if err != nil {
		return err
	}
	if len(samples) == 0 {
		return fmt.Errorf("hostmon has no samples yet")
	}
	if asTop && !status.Running {
		fmt.Fprintln(os.Stderr, "warning: sampler is stopped; showing the last persisted sample")
	}
	if asTop {
		for _, sample := range samples {
			fmt.Fprint(osStdout(), formatHostMonTop(sample))
		}
		return nil
	}
	for _, sample := range samples {
		fmt.Fprintln(osStdout(), formatHostMonLogLine(sample))
	}
	return nil
}

func formatHostMonStatus(status *client.HostMonStatus) string {
	run := "stopped"
	if status.Running {
		run = "running"
	}
	auto := "off"
	if status.AutoStart {
		auto = "on"
	}
	interval := status.IntervalSec
	if interval <= 0 {
		interval = 15
	}
	return fmt.Sprintf("hostmon     %s\nauto-start  %s\ninterval    %ds\nlog         %s (%s)\n",
		run, auto, interval, status.LogPath, formatHostMonBytes(status.LogBytes))
}

func formatHostMonTop(sample client.HostMonSample) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s  load %.1f  cpu %.0f%% user  %.0f%% iowait  disk %s read %.0f iops  %s  write %.0f iops\n",
		sample.T, sample.Load1, sample.CPU.User, sample.CPU.IOWait,
		emptyDash(sample.Disk.Dev), sample.Disk.RIops, formatRate(sample.Disk.RBps), sample.Disk.WIops)
	if len(sample.Top) == 0 {
		return b.String()
	}
	fmt.Fprintf(&b, "  PID    READ/s   WRITE/s  CPU%%  COMM\n")
	for _, proc := range sample.Top {
		fmt.Fprintf(&b, "  %-6d %-8s %-8s %-5.0f %s\n",
			proc.PID, formatRate(proc.ReadBps), formatRate(proc.WriteBps), proc.CPU, proc.Comm)
	}
	return b.String()
}

func formatHostMonLogLine(sample client.HostMonSample) string {
	line := fmt.Sprintf("%s load=%.1f iowait=%.0f%% rdiops=%.0f",
		sample.T, sample.Load1, sample.CPU.IOWait, sample.Disk.RIops)
	if len(sample.Top) > 0 {
		top := sample.Top[0]
		line += fmt.Sprintf(" top=%d %s read=%s/s", top.PID, top.Comm, formatRate(top.ReadBps))
	}
	return line
}

func formatRate(n float64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	switch {
	case n < 1024:
		return fmt.Sprintf("%s%.0fB", sign, n)
	case n < 1024*1024:
		return fmt.Sprintf("%s%.1fKiB", sign, n/1024)
	default:
		return fmt.Sprintf("%s%.1fMiB", sign, n/(1024*1024))
	}
}

func formatHostMonBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%d KiB", n/1024)
	}
	return fmt.Sprintf("%d MiB", n/(1024*1024))
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
