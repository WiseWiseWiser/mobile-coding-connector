package run

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/run/daemon"
	"github.com/xhd2015/ai-critic/server/config"
	"github.com/xhd2015/less-gen/flags"
)

const (
	sysctlUnitName = "ai-critic-server"
	sysctlUnitPath = "/etc/systemd/system/ai-critic-server.service"
)

// Test hooks (nil / zero → real OS).
var (
	integrationOut        io.Writer
	integrationErr        io.Writer
	integrationGOOS       = runtime.GOOS
	integrationExecutable = os.Executable
	integrationGetuid     = os.Getuid
	integrationReadFile   = os.ReadFile
	integrationWriteFile  = os.WriteFile
	integrationMkdirAll   = os.MkdirAll
	integrationHome       = os.Getenv("HOME")
	integrationUnitPath   = sysctlUnitPath
	integrationSystemctl  = defaultSystemctl
	integrationJournalctl = defaultJournalctl
	integrationListen     = daemon.IsPortListening
	integrationPing       = defaultPing
)

func runIntegrationSystemctl(args []string) error {
	if len(args) == 0 || isRunHelpToken(args[0]) {
		fmt.Fprint(integrationStdout(), strings.TrimPrefix(integrationSystemctlHelp, "\n"))
		return nil
	}
	f, rest, err := parseSystemctlFlags(args, integrationSystemctlHelp)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if len(rest) == 0 {
		fmt.Fprint(integrationStdout(), strings.TrimPrefix(integrationSystemctlHelp, "\n"))
		return nil
	}
	cmd := rest[0]
	cmdHelp := systemctlCommandHelp(cmd)
	f2, rest2, err := parseSystemctlFlags(rest[1:], cmdHelp)
	if err != nil {
		if errors.Is(err, flags.ErrHelp) {
			return nil
		}
		return err
	}
	if f2.DryRun {
		f.DryRun = true
	}
	if f2.Port > 0 {
		f.Port = f2.Port
	}
	if f2.Lines > 0 {
		f.Lines = f2.Lines
	}
	if f2.NoFollow {
		f.NoFollow = true
	}
	if len(rest2) > 0 && !isRunHelpToken(rest2[0]) {
		return fmt.Errorf("unexpected args: %s", strings.Join(rest2, " "))
	}
	if f.Port <= 0 {
		f.Port = config.DefaultServerPort
	}

	switch cmd {
	case "status":
		return sysctlStatus(f.Port)
	case "start":
		return sysctlStart(f.DryRun, f.Port)
	case "stop":
		return sysctlStop(f.DryRun)
	case "logs":
		return sysctlLogs(f.DryRun, f.Lines, !f.NoFollow)
	case "show-config":
		return sysctlShowConfig(f.Port)
	case "-h", "--help", "help":
		fmt.Fprint(integrationStdout(), strings.TrimPrefix(integrationSystemctlHelp, "\n"))
		return nil
	default:
		return fmt.Errorf("unknown systemctl command %q (want status|start|stop|logs|show-config)", cmd)
	}
}

func systemctlCommandHelp(cmd string) string {
	switch cmd {
	case "status":
		return "Usage: ai-critic integration systemctl status [--port N]\n\nShow systemd status and /ping.\n"
	case "start":
		return "Usage: ai-critic integration systemctl start [--port N] [--dry-run]\n\nEnsure unit file, enable, start (restart if already active).\n"
	case "stop":
		return "Usage: ai-critic integration systemctl stop [--dry-run]\n\nStop the systemd unit (does not disable).\n"
	case "show-config":
		return "Usage: ai-critic integration systemctl show-config [--port N]\n\nPrint generated unit, on-disk match, and day-to-day commands.\n"
	case "logs":
		return "Usage: ai-critic integration systemctl logs [--lines N] [--no-follow] [--dry-run]\n\nFollow journalctl -u ai-critic-server (default -n 100 -f --no-pager).\n"
	default:
		return integrationSystemctlHelp
	}
}

func requireLinux() error {
	if integrationGOOS != "linux" {
		return fmt.Errorf("systemctl integration is Linux-only")
	}
	return nil
}

func requireRoot() error {
	if integrationGetuid() != 0 {
		return fmt.Errorf("need root to write %s", integrationUnitPath)
	}
	return nil
}

func resolveInstall(port int) (bin, workdir, home, unit string, err error) {
	bin, err = integrationExecutable()
	if err != nil {
		return "", "", "", "", err
	}
	bin, err = filepath.Abs(bin)
	if err != nil {
		return "", "", "", "", err
	}
	if resolved, rerr := filepath.EvalSymlinks(bin); rerr == nil {
		bin = resolved
	}
	workdir = filepath.Dir(bin)
	home = strings.TrimSpace(integrationHome)
	if home == "" {
		home = "/root"
	}
	unit = generateSystemdUnit(bin, workdir, home, port)
	return bin, workdir, home, unit, nil
}

func generateSystemdUnit(bin, workdir, home string, port int) string {
	if port <= 0 {
		port = config.DefaultServerPort
	}
	if home == "" {
		home = "/root"
	}
	execStart := bin
	if strings.ContainsAny(bin, " \t") {
		execStart = `"` + bin + `"`
	}
	return fmt.Sprintf(`[Unit]
Description=ai-critic-server
After=network.target

[Service]
User=root
WorkingDirectory=%s
Environment=HOME=%s
ExecStart=%s --port %d
Restart=always
RestartSec=2
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`, workdir, home, execStart, port)
}

func normalizeUnit(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimSpace(s) + "\n"
}

func unitsMatch(a, b string) bool {
	return normalizeUnit(a) == normalizeUnit(b)
}

func readOnDiskUnit() (body string, exists bool, err error) {
	data, err := integrationReadFile(integrationUnitPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(data), true, nil
}

func onDiskState(generated string) string {
	body, exists, err := readOnDiskUnit()
	if err != nil {
		return "unreadable"
	}
	if !exists {
		return "missing"
	}
	if unitsMatch(body, generated) {
		return "match"
	}
	return "differ"
}

func sysctlShowConfig(port int) error {
	if err := requireLinux(); err != nil {
		return err
	}
	bin, workdir, home, unit, err := resolveInstall(port)
	if err != nil {
		return err
	}
	state := onDiskState(unit)
	data := ".ai-critic"
	if home != "" {
		data = filepath.Join(home, ".ai-critic")
	}
	out := integrationStdout()
	fmt.Fprintf(out, "unit:       %s\n", integrationUnitPath)
	fmt.Fprintf(out, "binary:     %s\n", bin)
	fmt.Fprintf(out, "workdir:    %s\n", workdir)
	fmt.Fprintf(out, "port:       %d\n", port)
	fmt.Fprintf(out, "home:       %s\n", home)
	fmt.Fprintf(out, "data:       %s\n", data)
	fmt.Fprintf(out, "on-disk:    %s\n", state)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "# unit")
	fmt.Fprint(out, normalizeUnit(unit))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "# day-to-day")
	self := bin
	fmt.Fprintf(out, "%s integration systemctl status\n", self)
	fmt.Fprintf(out, "%s integration systemctl start\n", self)
	fmt.Fprintf(out, "%s integration systemctl stop\n", self)
	fmt.Fprintf(out, "%s integration systemctl logs\n", self)
	fmt.Fprintf(out, "journalctl -u %s -n 100 -f --no-pager\n", sysctlUnitName)
	fmt.Fprintf(out, "curl -sS http://127.0.0.1:%d/ping\n", port)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "# replace binary then:")
	fmt.Fprintf(out, "scp ai-critic-server-linux-amd64 host:%s\n", bin)
	fmt.Fprintf(out, "%s integration systemctl start\n", self)
	return nil
}

func sysctlStart(dryRun bool, port int) error {
	if err := requireLinux(); err != nil {
		return err
	}
	if !dryRun {
		if err := requireRoot(); err != nil {
			return err
		}
	}
	_, _, _, unit, err := resolveInstall(port)
	if err != nil {
		return err
	}
	state := onDiskState(unit)
	out := integrationStdout()
	needWrite := state != "match"

	if dryRun {
		if needWrite {
			fmt.Fprintf(out, "would write %s\n", integrationUnitPath)
			fmt.Fprintln(out, "would systemctl daemon-reload")
		}
		fmt.Fprintf(out, "would systemctl enable %s\n", sysctlUnitName)
		fmt.Fprintf(out, "would systemctl restart %s\n", sysctlUnitName)
		return nil
	}

	if needWrite {
		if err := integrationMkdirAll(filepath.Dir(integrationUnitPath), 0o755); err != nil {
			return err
		}
		if err := integrationWriteFile(integrationUnitPath, []byte(normalizeUnit(unit)), 0o644); err != nil {
			return fmt.Errorf("need root to write %s: %w", integrationUnitPath, err)
		}
		if _, err := integrationSystemctl("daemon-reload"); err != nil {
			return fmt.Errorf("systemctl daemon-reload: %w", err)
		}
	}
	fmt.Fprintf(out, "unit:     %s  %s\n", stateWord(state, needWrite), integrationUnitPath)

	warnKeepAlive(port)

	if _, err := integrationSystemctl("enable", sysctlUnitName); err != nil {
		return fmt.Errorf("systemctl enable: %w", err)
	}
	fmt.Fprintln(out, "enable:   ok")

	if _, err := integrationSystemctl("restart", sysctlUnitName); err != nil {
		return fmt.Errorf("systemctl restart: %w", err)
	}
	active, pid := sysctlActivePID()
	fmt.Fprintf(out, "start:    %s", active)
	if pid != "" && pid != "0" {
		fmt.Fprintf(out, "  pid=%s", pid)
	}
	fmt.Fprintln(out)

	code, body, perr := integrationPing(port)
	if perr != nil {
		fmt.Fprintf(out, "ping:     error %v\n", perr)
		return fmt.Errorf("ping http://127.0.0.1:%d/ping: %w", port, perr)
	}
	fmt.Fprintf(out, "ping:     %d %s\n", code, strings.TrimSpace(body))
	if code != 200 {
		return fmt.Errorf("ping http://127.0.0.1:%d/ping: HTTP %d", port, code)
	}
	return nil
}

func stateWord(state string, wrote bool) string {
	if wrote {
		if state == "missing" {
			return "created"
		}
		return "updated"
	}
	return "match"
}

func sysctlStop(dryRun bool) error {
	if err := requireLinux(); err != nil {
		return err
	}
	out := integrationStdout()
	if dryRun {
		fmt.Fprintf(out, "would systemctl stop %s\n", sysctlUnitName)
		return nil
	}
	if err := requireRoot(); err != nil {
		return err
	}
	if _, err := integrationSystemctl("stop", sysctlUnitName); err != nil {
		return fmt.Errorf("systemctl stop: %w", err)
	}
	active, _ := sysctlActivePID()
	fmt.Fprintf(out, "stop:     %s\n", active)
	return nil
}

func sysctlStatus(port int) error {
	if err := requireLinux(); err != nil {
		return err
	}
	out := integrationStdout()
	text, err := integrationSystemctl("status", sysctlUnitName, "--no-pager")
	if looksUnitNotFound(text, err) {
		return fmt.Errorf("systemctl: Unit %s.service not found", sysctlUnitName)
	}
	fmt.Fprint(out, text)
	if text != "" && !strings.HasSuffix(text, "\n") {
		fmt.Fprintln(out)
	}
	code, body, perr := integrationPing(port)
	if perr != nil {
		fmt.Fprintf(out, "ping:     error %v\n", perr)
		return nil
	}
	fmt.Fprintf(out, "ping:     %d %s\n", code, strings.TrimSpace(body))
	return nil
}

func looksUnitNotFound(text string, err error) bool {
	s := strings.ToLower(text + " " + errString(err))
	return strings.Contains(s, "not found") || strings.Contains(s, "could not be found")
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func sysctlActivePID() (active, pid string) {
	active = "unknown"
	if v, err := integrationSystemctl("show", sysctlUnitName, "-p", "ActiveState", "--value"); err == nil {
		active = strings.TrimSpace(v)
	}
	if v, err := integrationSystemctl("show", sysctlUnitName, "-p", "MainPID", "--value"); err == nil {
		pid = strings.TrimSpace(v)
	}
	return active, pid
}

func warnKeepAlive(port int) {
	if integrationListen == nil {
		return
	}
	if integrationListen(config.KeepAlivePort) {
		fmt.Fprintf(integrationStderr(), "warning: keep-alive looks running on :%d; systemd and keep-alive should not both own :%d\n", config.KeepAlivePort, port)
	}
}

func journalctlArgs(lines int, follow bool) []string {
	if lines <= 0 {
		lines = 100
	}
	args := []string{"-u", sysctlUnitName, "-n", fmt.Sprint(lines), "--no-pager"}
	if follow {
		args = append(args, "-f")
	}
	return args
}

func sysctlLogs(dryRun bool, lines int, follow bool) error {
	if err := requireLinux(); err != nil {
		return err
	}
	args := journalctlArgs(lines, follow)
	if dryRun {
		fmt.Fprintf(integrationStdout(), "would journalctl %s\n", strings.Join(args, " "))
		return nil
	}
	return integrationJournalctl(args)
}

func defaultSystemctl(args ...string) (string, error) {
	cmd := exec.Command("systemctl", args...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}

func defaultJournalctl(args []string) error {
	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = integrationStdout()
	cmd.Stderr = integrationStderr()
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func defaultPing(port int) (int, string, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/ping", port))
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(body), nil
}
