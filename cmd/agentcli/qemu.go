package agentcli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/xhd2015/ai-critic/client"
	serverqemu "github.com/xhd2015/ai-critic/server/qemu"
	shared "github.com/xhd2015/dot-pkgs/go-pkgs/qemu"
	"github.com/xhd2015/less-gen/flags"
)

const qemuHelp = `Usage: %s qemu <command> [OPTIONS]

Manage a QEMU guest used for optional in-guest cloudflared.

Commands:
  status     qemu pid, guest ssh, disk
  doctor     verbose diagnose + hints
  start      idempotent bring-up
  stop       stop qemu (overlay kept)
  restart    stop then start
  purge      stop + delete overlay (--deep backing too)
  show       topology / paths / ports
  logs       tail serial.log
  sh         interactive ssh into the guest (local-agent only)
  exec <cmd> [args...]   run command in the guest
  cloudflared …          guest cloudflared (status/start/login/…)
  config                 show/set ~/.ai-critic/qemu.json enabled flag

Options (before command where applicable):
  --dry-run
  --yes                  (purge)
  --deep                 (purge: also delete backing / cert)
  --url URL              cloudflared origin
  --lines N
  -h, --help

Guest state: ~/.ai-critic/qemu/*
Config:      ~/.ai-critic/qemu.json  ({"enabled": false})

Examples:
  %s qemu status
  %s qemu start --dry-run
  %s qemu cloudflared status
  %s qemu config --enabled
`

const qemuCFHelp = `Usage: %s qemu cloudflared <command> [OPTIONS]

Guest cloudflared inside the qemu guest.

Commands:
  status     guest ssh + cf pid + public url + cert
  doctor     status + log tail + hints
  start      idempotent quick tunnel (--url)
  stop       kill guest cloudflared (qemu kept)
  restart    stop then start
  purge      stop + logs/pid/url; --deep also cert.pem
  show       topology / paths
  logs       tail guest cloudflared log
  login      print CF dash URL; wait for cert.pem
  list       cloudflared tunnel list (needs cert)

Options:
  --url URL
  --dry-run
  --yes
  --deep
  --lines N
  -h, --help
`

type qemuFlags struct {
	DryRun bool
	Yes    bool
	Deep   bool
	Lines  int
	URL    string
}

func runQemu(resolve func() (*client.Client, error), args []string) error {
	help := fmt.Sprintf(qemuHelp, active.Name, active.Name, active.Name, active.Name, active.Name)
	if len(args) == 0 {
		fmt.Print(help)
		return nil
	}
	if isHelpToken(args[0]) && len(args) == 1 {
		fmt.Print(help)
		return nil
	}

	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "cloudflared":
		return runQemuCloudflared(resolve, rest)
	case "config":
		return runQemuConfig(resolve, rest)
	case "sh":
		return runQemuSh(resolve, rest)
	case "exec":
		return runQemuExec(resolve, rest)
	case "-h", "--help", "help":
		fmt.Print(help)
		return nil
	default:
		f, rem, err := parseQemuFlags(rest)
		if err != nil {
			return err
		}
		if len(rem) > 0 {
			return fmt.Errorf("%s takes no args (got %v)", cmd, rem)
		}
		return runQemuAction(resolve, cmd, f)
	}
}

func parseQemuFlags(args []string) (qemuFlags, []string, error) {
	var f qemuFlags
	rest, err := flags.
		Bool("--dry-run", &f.DryRun).
		Bool("--yes", &f.Yes).
		Bool("--deep", &f.Deep).
		Int("--lines", &f.Lines).
		String("--url", &f.URL).
		HelpNoExit().
		HelpFunc("-h,--help", func() {}).
		Parse(args)
	if err != nil {
		return f, nil, err
	}
	return f, rest, nil
}

func runQemuAction(resolve func() (*client.Client, error), action string, f qemuFlags) error {
	switch action {
	case "status", "doctor", "start", "stop", "restart", "purge", "show", "logs":
	default:
		return fmt.Errorf("unknown qemu command %q (run: %s qemu -h)", action, active.Name)
	}
	if action == "purge" && !f.Yes && !f.DryRun {
		return fmt.Errorf("purge requires --yes")
	}

	if active.Name == "local-agent" {
		return runQemuLocal(action, f)
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	resp, err := cli.QemuAction(action, serverqemu.ActionRequest{
		DryRun: f.DryRun,
		Yes:    f.Yes,
		Deep:   f.Deep,
		Lines:  f.Lines,
		URL:    f.URL,
	})
	if err != nil {
		return err
	}
	return printQemuResp(resp)
}

func runQemuLocal(action string, f qemuFlags) error {
	m := &shared.Manager{
		Cfg:    shared.AiCriticConfig(),
		Host:   shared.LocalHost{},
		DryRun: f.DryRun,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	switch action {
	case "show":
		fmt.Print(localShowText(m.Cfg))
		return nil
	case "status":
		kv, err := m.Status()
		printLocalStatus(kv)
		if err != nil {
			return err
		}
		if kv["qemu_alive"] != "yes" {
			return fmt.Errorf("qemu: down")
		}
		return nil
	case "doctor":
		kv, err := m.Status()
		printLocalStatus(kv)
		lines := f.Lines
		if lines <= 0 {
			lines = 40
		}
		if !f.DryRun {
			if logs, lerr := m.Logs(lines); lerr == nil {
				fmt.Println("serial:")
				fmt.Print(ensureNL(logs))
			} else {
				fmt.Fprintf(os.Stderr, "warning: serial: %v\n", lerr)
			}
		}
		fmt.Println("hint: guest state under ~/.ai-critic/qemu")
		fmt.Println("hint: enable guest CF with: echo '{\"enabled\":true}' > ~/.ai-critic/qemu.json")
		if err != nil {
			return err
		}
		if kv["qemu_alive"] != "yes" {
			return fmt.Errorf("qemu: down")
		}
		return nil
	case "start":
		kv, err := m.Start()
		if f.DryRun {
			return err
		}
		if kv["skip"] == "running" {
			fmt.Fprintln(os.Stderr, "warning: already running; not recreating disk")
		}
		printLocalStatus(kv)
		return err
	case "stop":
		return m.Stop()
	case "restart":
		kv, err := m.Restart()
		if f.DryRun {
			return err
		}
		printLocalStatus(kv)
		return err
	case "purge":
		return m.Purge(f.Deep)
	case "logs":
		lines := f.Lines
		if lines <= 0 {
			lines = 80
		}
		out, err := m.Logs(lines)
		if out != "" {
			fmt.Print(ensureNL(out))
		}
		return err
	default:
		return fmt.Errorf("unknown qemu command %q", action)
	}
}

func runQemuCloudflared(resolve func() (*client.Client, error), args []string) error {
	help := fmt.Sprintf(qemuCFHelp, active.Name)
	if len(args) == 0 || isHelpToken(args[0]) {
		fmt.Print(help)
		return nil
	}
	action := args[0]
	f, rem, err := parseQemuFlags(args[1:])
	if err != nil {
		return err
	}
	if len(rem) > 0 {
		return fmt.Errorf("%s takes no args (got %v)", action, rem)
	}
	if action == "purge" && !f.Yes && !f.DryRun {
		return fmt.Errorf("purge requires --yes")
	}

	if active.Name == "local-agent" {
		return runQemuCFLocal(action, f)
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	resp, err := cli.QemuCloudflaredAction(action, serverqemu.ActionRequest{
		DryRun: f.DryRun,
		Yes:    f.Yes,
		Deep:   f.Deep,
		Lines:  f.Lines,
		URL:    f.URL,
	})
	if err != nil {
		return err
	}
	return printQemuResp(resp)
}

func runQemuCFLocal(action string, f qemuFlags) error {
	m := &shared.Manager{
		Cfg:    shared.AiCriticConfig(),
		Host:   shared.LocalHost{},
		DryRun: f.DryRun,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	origin := f.URL
	switch action {
	case "show":
		fmt.Print(localShowCFText(m.Cfg, origin))
		return nil
	case "status":
		kv, err := m.CFStatus(origin)
		printLocalCFStatus(kv)
		if err != nil {
			return err
		}
		if kv["cf_alive"] != "yes" {
			return fmt.Errorf("cloudflared: down")
		}
		return nil
	case "doctor":
		kv, err := m.CFStatus(origin)
		printLocalCFStatus(kv)
		lines := f.Lines
		if lines <= 0 {
			lines = 40
		}
		if !f.DryRun {
			if logs, lerr := m.CFLogs(lines); lerr == nil {
				fmt.Println("log:")
				fmt.Print(ensureNL(logs))
			}
		}
		if err != nil {
			return err
		}
		if kv["cf_alive"] != "yes" {
			return fmt.Errorf("cloudflared: down")
		}
		return nil
	case "start":
		kv, err := m.CFStart(origin)
		if f.DryRun {
			return err
		}
		printLocalCFStatus(kv)
		return err
	case "stop":
		return m.CFStop()
	case "restart":
		kv, err := m.CFRestart(origin)
		if f.DryRun {
			return err
		}
		printLocalCFStatus(kv)
		return err
	case "purge":
		return m.CFPurge(f.Deep)
	case "logs":
		lines := f.Lines
		if lines <= 0 {
			lines = 80
		}
		out, err := m.CFLogs(lines)
		if out != "" {
			fmt.Print(ensureNL(out))
		}
		return err
	case "login":
		kv, err := m.CFLogin()
		printLocalCFStatus(kv)
		return err
	case "list":
		out, err := m.CFList()
		if out != "" {
			fmt.Print(ensureNL(out))
		}
		return err
	default:
		return fmt.Errorf("unknown qemu cloudflared command %q", action)
	}
}

func runQemuConfig(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 && isHelpToken(args[0]) {
		fmt.Printf("Usage: %s qemu config [--enabled|--disabled]\n\nShow or set ~/.ai-critic/qemu.json enabled flag.\n", active.Name)
		return nil
	}
	var enabled, disabled bool
	rest, err := flags.
		Bool("--enabled", &enabled).
		Bool("--disabled", &disabled).
		HelpNoExit().
		HelpFunc("-h,--help", func() {}).
		Parse(args)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return fmt.Errorf("config takes no args (got %v)", rest)
	}
	if enabled && disabled {
		return fmt.Errorf("--enabled and --disabled are mutually exclusive")
	}

	if active.Name == "local-agent" {
		path := shared.DefaultFilePath(mustHome())
		if enabled || disabled {
			cfg := shared.FileConfig{Enabled: enabled}
			if err := shared.SaveFileConfig(path, cfg); err != nil {
				return err
			}
		}
		cfg, err := shared.LoadFileConfig(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			cfg = shared.FileConfig{}
		}
		fmt.Printf("path:     %s\n", path)
		fmt.Printf("enabled:  %v\n", cfg.Enabled)
		fmt.Printf("guest:    %s\n", shared.AiCriticConfig().Dir)
		return nil
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	if enabled || disabled {
		out, err := cli.QemuSetConfig(enabled)
		if err != nil {
			return err
		}
		fmt.Printf("enabled:  %v\n", out.Enabled)
		return nil
	}
	out, err := cli.QemuGetConfig()
	if err != nil {
		return err
	}
	fmt.Printf("enabled:  %v\n", out.Enabled)
	return nil
}

func runQemuSh(resolve func() (*client.Client, error), args []string) error {
	f, rem, err := parseQemuFlags(args)
	if err != nil {
		return err
	}
	if len(rem) > 0 {
		return fmt.Errorf("sh takes no args (got %v)", rem)
	}
	if active.Name != "local-agent" {
		return fmt.Errorf("qemu sh is local-agent only; use: %s qemu exec -- bash -l", active.Name)
	}
	cfg := shared.AiCriticConfig()
	sshArgs := shared.GuestSSHArgs(cfg, true)
	if f.DryRun {
		fmt.Printf("[dry-run] would %s\n", strings.Join(sshArgs, " "))
		return nil
	}
	cmd := exec.Command(sshArgs[0], sshArgs[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runQemuExec(resolve func() (*client.Client, error), args []string) error {
	f, rem, err := parseQemuFlags(args)
	if err != nil {
		return err
	}
	rem = stripLeadingDashDash(rem)
	if len(rem) == 0 {
		return fmt.Errorf("exec requires a command")
	}

	if active.Name == "local-agent" {
		cfg := shared.AiCriticConfig()
		sshArgs := shared.GuestSSHArgs(cfg, false)
		full := append(sshArgs, rem...)
		if f.DryRun {
			fmt.Printf("[dry-run] would %s\n", strings.Join(full, " "))
			return nil
		}
		cmd := exec.Command(full[0], full[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	cli, err := resolve()
	if err != nil {
		return err
	}
	resp, err := cli.QemuAction("exec", serverqemu.ActionRequest{
		DryRun: f.DryRun,
		Argv:   rem,
	})
	if err != nil {
		return err
	}
	return printQemuResp(resp)
}

func printQemuResp(resp *serverqemu.ActionResponse) error {
	if resp == nil {
		return fmt.Errorf("empty response")
	}
	if resp.Output != "" {
		fmt.Print(ensureNL(resp.Output))
	} else if len(resp.KV) > 0 {
		printLocalStatus(resp.KV)
	}
	if !resp.OK {
		if resp.Error != "" {
			return fmt.Errorf("%s", resp.Error)
		}
		return fmt.Errorf("qemu action failed")
	}
	return nil
}

func printLocalStatus(kv map[string]string) {
	alive := kv["qemu_alive"] == "yes" || kv["skip"] == "running"
	cfg := shared.AiCriticConfig()
	if alive {
		if pid := kv["qemu_pid"]; pid != "" {
			fmt.Printf("qemu:      running  pid=%s  %s\n", pid, cfg.Accel)
		} else {
			fmt.Printf("qemu:      running  %s\n", cfg.Accel)
		}
	} else if len(kv) > 0 || kv != nil {
		fmt.Println("qemu:      down")
	}
	if len(kv) == 0 {
		return
	}
	fmt.Printf("ssh:       127.0.0.1:%d\n", cfg.SSHPort)
	switch kv["guest_ssh"] {
	case "ok":
		fmt.Printf("guest:     ssh ok  %s@qemu-guest\n", cfg.User)
	case "wait":
		fmt.Println("guest:     ssh wait (booting)")
	default:
		if kv["guest_ssh"] != "" {
			fmt.Println("guest:     ssh fail")
		}
	}
	if kv["overlay_bytes"] != "" || kv["backing_bytes"] != "" {
		fmt.Printf("disk:      overlay %s  backing %s  %s\n",
			shared.HumanBytes(kv["overlay_bytes"]), shared.HumanBytes(kv["backing_bytes"]), cfg.Dir)
	}
}

func printLocalCFStatus(kv map[string]string) {
	if kv["cf_alive"] == "yes" {
		fmt.Printf("cloudflared: running  pid=%s\n", kv["cf_pid"])
	} else if len(kv) > 0 {
		fmt.Println("cloudflared: down")
	}
	if u := kv["url"]; u != "" {
		fmt.Printf("url:         %s\n", u)
	}
	if c := kv["cert"]; c != "" {
		fmt.Printf("cert:        %s\n", c)
	}
}

func localShowText(cfg shared.Config) string {
	c := shared.AiCriticConfig()
	path := shared.DefaultFilePath(mustHome())
	var b strings.Builder
	b.WriteString("kind:       ai-critic qemu guest\n")
	fmt.Fprintf(&b, "dir:        %s\n", c.Dir)
	fmt.Fprintf(&b, "user:       %s\n", c.User)
	fmt.Fprintf(&b, "ssh:        127.0.0.1:%d\n", c.SSHPort)
	fmt.Fprintf(&b, "accel:      %s\n", c.Accel)
	fmt.Fprintf(&b, "mem/cpus:   %dMB / %d\n", c.MemMB, c.CPUs)
	fmt.Fprintf(&b, "config:     %s  enabled=%v\n", path, shared.IsEnabled(path))
	return b.String()
}

func localShowCFText(cfg shared.Config, origin string) string {
	c := shared.AiCriticConfig()
	if origin == "" {
		origin = c.CFOriginURL
	}
	var b strings.Builder
	b.WriteString("kind:       guest cloudflared\n")
	fmt.Fprintf(&b, "origin:     %s\n", origin)
	fmt.Fprintf(&b, "bin:        %s\n", c.CFBin)
	fmt.Fprintf(&b, "cert:       %s\n", c.CFCert)
	fmt.Fprintf(&b, "guest dir:  %s\n", c.Dir)
	return b.String()
}

func mustHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

func ensureNL(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

func stripLeadingDashDash(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}
	return args
}

func isHelpToken(s string) bool {
	return s == "-h" || s == "--help" || s == "help"
}
