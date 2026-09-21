package agentcli

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/cmd/agentcli/gomodrelay"
	servergomod "github.com/xhd2015/ai-critic/server/gomod"
	"github.com/xhd2015/less-gen/flags"
)

const goHelp = `Usage: %s go <command> [OPTIONS]

Go tooling helpers backed by the remote agent.

Commands:
  mod-proxy         GOPROXY server on the remote agent (serves cache/download)
  mod-proxy-relay   local fast-fail relay; 404s when upstream is down so the chain falls through

Options:
  -h, --help    show this help
`

const goModProxyHelp = `Usage: %s go mod-proxy <command> [OPTIONS]

GOPROXY server on the remote agent server (default port 21000, no token).
Serves the on-disk cache/download tree; 404s let the GOPROXY chain fall through.

Commands:
  status     show runtime + config state
  start      start serving now
  stop       stop serving
  enable     auto-start on agent server boot (use --now to also start)
  disable    clear auto-start (use --now to also stop)
  logs       tail the service log

Options:
  --port N         listen port (default 21000)
  --root DIR       cache/download data root (default /root/gomod-proxy/cache/download)
  --now            with enable/disable: also start/stop
  --lines N        logs: number of lines (default 50)
  -h, --help       show this help
`

const goRelayHelp = `Usage: %s go mod-proxy-relay <command> [OPTIONS]

Local fast-fail GOPROXY relay on 127.0.0.1:21001, hosted by the local-agent
macOS app keep-alive daemon. When the upstream is unreachable the relay
answers 404 so the GOPROXY chain falls through. Use a direct IP for --upstream
(e.g. http://10.91.186.143:21000), not a Cloudflare hostname.

Commands:
  status     show relay state, upstream health, and GOPROXY usage
  start      start the relay now
  stop       stop the relay
  enable     auto-start on daemon boot (use --now to also start)
  disable    clear auto-start (use --now to also stop)
  logs       tail the relay log

Options:
  --port PORT            local listen port (default 21001)
  --upstream URL         upstream GOPROXY base URL (required on first start/enable)
  --dial-timeout D       upstream connect timeout (default 2s)
  --down-recheck D       re-probe interval while down (default 10s)
  --now                  with enable/disable: also start/stop
  --lines N              logs: number of lines (default 50)
  --foreground           debug: run a standalone relay in this terminal (no daemon)
  -h, --help             show this help

GOPROXY recipe (after enable --now):
  export GOPROXY=http://127.0.0.1:21001,https://proxy.golang.org,direct
  export GOINSECURE=127.0.0.1
`

type gomodFlags struct {
	Now         bool
	Port        int
	Root        string
	Lines       int
	Upstream    string
	DialTimeout string
	DownRecheck string
	Foreground  bool
}

func runGo(resolve func() (*client.Client, error), args []string) error {
	help := fmt.Sprintf(goHelp, active.Name)
	if len(args) == 0 || (isHelpToken(args[0]) && len(args) == 1) {
		fmt.Print(help)
		return nil
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "mod-proxy":
		return runGomodProxy(resolve, rest)
	case "mod-proxy-relay":
		return runGomodRelay(rest)
	case "-h", "--help", "help":
		fmt.Print(help)
		return nil
	default:
		return fmt.Errorf("unknown go command %q (run: %s go -h)", cmd, active.Name)
	}
}

func parseGomodFlags(args []string) (gomodFlags, []string, error) {
	var f gomodFlags
	rest, err := flags.
		Bool("--now", &f.Now).
		Int("--port", &f.Port).
		String("--root", &f.Root).
		Int("--lines", &f.Lines).
		String("--upstream", &f.Upstream).
		String("--dial-timeout", &f.DialTimeout).
		String("--down-recheck", &f.DownRecheck).
		Bool("--foreground", &f.Foreground).
		HelpNoExit().
		HelpFunc("-h,--help", func() {}).
		Parse(args)
	if err != nil {
		return f, nil, err
	}
	return f, rest, nil
}

func runGomodProxy(resolve func() (*client.Client, error), args []string) error {
	help := fmt.Sprintf(goModProxyHelp, active.Name)
	if len(args) == 0 || (isHelpToken(args[0]) && len(args) == 1) {
		fmt.Print(help)
		return nil
	}
	action := args[0]
	rest := args[1:]
	switch action {
	case "status", "start", "stop", "enable", "disable", "logs":
	default:
		return fmt.Errorf("unknown mod-proxy command %q (run: %s go mod-proxy -h)", action, active.Name)
	}
	f, _, err := parseGomodFlags(rest)
	if err != nil {
		return err
	}
	cli, err := resolve()
	if err != nil {
		return err
	}
	resp, err := cli.GomodAction(action, servergomod.ActionRequest{
		Now:   f.Now,
		Port:  f.Port,
		Root:  f.Root,
		Lines: f.Lines,
	})
	if err != nil {
		return err
	}
	return printGomodResult(resp.OK, resp.KV, resp.Output, resp.Error)
}

// runGomodRelay manages the local relay via the daemon control socket.
func runGomodRelay(args []string) error {
	help := fmt.Sprintf(goRelayHelp, active.Name)
	if len(args) == 0 || (isHelpToken(args[0]) && len(args) == 1) {
		fmt.Print(help)
		return nil
	}
	action := args[0]
	rest := args[1:]
	switch action {
	case "status", "start", "stop", "enable", "disable", "logs":
	default:
		return fmt.Errorf("unknown mod-proxy-relay command %q (run: %s go mod-proxy-relay -h)", action, active.Name)
	}
	f, _, err := parseGomodFlags(rest)
	if err != nil {
		return err
	}

	if action == "logs" {
		return tailRelayLog(f.Lines)
	}
	if f.Foreground && (action == "start" || (action == "enable" && f.Now)) {
		return runRelayForeground(f)
	}

	base, err := gomodrelay.BaseDir()
	if err != nil {
		return err
	}
	req := gomodrelay.OpRequest{Op: action, Now: f.Now, Port: f.Port, Upstream: f.Upstream}
	if f.DialTimeout != "" {
		d, perr := time.ParseDuration(f.DialTimeout)
		if perr != nil {
			return fmt.Errorf("invalid --dial-timeout %q: %w", f.DialTimeout, perr)
		}
		req.DialTimeoutMS = int(d.Milliseconds())
	}
	if f.DownRecheck != "" {
		d, perr := time.ParseDuration(f.DownRecheck)
		if perr != nil {
			return fmt.Errorf("invalid --down-recheck %q: %w", f.DownRecheck, perr)
		}
		req.DownRecheckMS = int(d.Milliseconds())
	}
	resp, err := gomodrelay.ControlClient(gomodrelay.SocketPath(base), req)
	if err != nil {
		return err
	}
	if err := printGomodResult(resp.OK, resp.KV, resp.Output, resp.Error); err != nil {
		return err
	}
	if resp.OK && (action == "status" || action == "start" || (action == "enable" && f.Now)) {
		running := resp.KV["status"] == "running"
		if action == "status" || running {
			fmt.Print(formatRelayUsage(relayPort(resp.KV), running))
		}
	}
	return nil
}

func relayPort(kv map[string]string) string {
	if kv != nil {
		if p := kv["port"]; p != "" {
			return p
		}
	}
	return "21001"
}

// formatRelayUsage is the copy-paste GOPROXY demo appended after status/start.
func formatRelayUsage(port string, running bool) string {
	if port == "" {
		port = "21001"
	}
	base := fmt.Sprintf("http://127.0.0.1:%s", port)
	var b strings.Builder
	b.WriteByte('\n')
	if running {
		b.WriteString("Usage (GOPROXY via this relay):\n")
	} else {
		b.WriteString("Usage (start first: " + active.Name + " go mod-proxy-relay start)\n")
	}
	b.WriteString("  export GOPROXY=" + base + ",https://proxy.golang.org,direct\n")
	b.WriteString("  export GOINSECURE=127.0.0.1\n")
	if running {
		b.WriteString("\n  # smoke\n")
		b.WriteString("  curl -s " + base + "/github.com/!burnt!sushi/toml/@v/list\n")
		b.WriteString("\n  # fetch\n")
		b.WriteString("  go mod download github.com/google/uuid@v1.6.0\n")
	}
	return b.String()
}

// runRelayForeground runs a standalone relay in this terminal (debug mode).
func runRelayForeground(f gomodFlags) error {
	base, err := gomodrelay.BaseDir()
	if err != nil {
		return err
	}
	cfg, err := gomodrelay.LoadConfig(gomodrelay.ConfigPath(base))
	if err != nil {
		return err
	}
	if f.Port > 0 {
		cfg.Port = f.Port
	}
	if f.Upstream != "" {
		cfg.Upstream = f.Upstream
	}
	if f.DialTimeout != "" {
		d, perr := time.ParseDuration(f.DialTimeout)
		if perr != nil {
			return fmt.Errorf("invalid --dial-timeout %q: %w", f.DialTimeout, perr)
		}
		cfg.DialTimeoutMS = int(d.Milliseconds())
	}
	if f.DownRecheck != "" {
		d, perr := time.ParseDuration(f.DownRecheck)
		if perr != nil {
			return fmt.Errorf("invalid --down-recheck %q: %w", f.DownRecheck, perr)
		}
		cfg.DownRecheckMS = int(d.Milliseconds())
	}
	if cfg.Upstream == "" {
		return fmt.Errorf("--upstream is required (e.g. --upstream http://10.91.186.143:21000)")
	}
	color := stderrColorEnabled()
	logf := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)
		if color {
			msg = colorLabel(msg)
		}
		fmt.Fprintf(os.Stderr, "%s\n", msg)
	}
	relay := gomodrelay.NewRelay(cfg, logf)
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.EffectivePort())
	ln, lerr := net.Listen("tcp", addr)
	if lerr != nil {
		return lerr
	}
	fmt.Fprintf(os.Stderr, "relaying (Ctrl-C to stop)  %s -> %s\n", addr, cfg.Upstream)
	ok, elapsed, perr := relay.ProbeUp()
	if ok {
		fmt.Fprintf(os.Stderr, "  Health     up (probed in %s)\n", elapsed.Round(time.Millisecond))
	} else {
		msg := fmt.Sprintf("warning: upstream %s unreachable (%v); marked down, re-checking every %s", cfg.Upstream, perr, cfg.EffectiveDownRecheck())
		if color {
			msg = colorLabel(msg)
		}
		fmt.Fprintf(os.Stderr, "%s\n", msg)
	}
	srv := &http.Server{Handler: relay}
	go func() { _ = srv.Serve(ln) }()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	relay.Close()
	_ = srv.Close()
	fmt.Fprintln(os.Stderr, "Stopped mod-proxy-relay")
	return nil
}

func tailRelayLog(n int) error {
	base, err := gomodrelay.BaseDir()
	if err != nil {
		return err
	}
	if n <= 0 {
		n = 50
	}
	data, err := os.ReadFile(gomodrelay.LogPath(base))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for _, l := range lines {
		fmt.Println(l)
	}
	return nil
}

// stderrColorEnabled implements the color recipe's auto mode for stderr.
func stderrColorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// colorLabel colors the leading "warning:"/"Error:" label of a message.
func colorLabel(msg string) string {
	for _, prefix := range []string{"warning:", "Error:"} {
		if strings.HasPrefix(msg, prefix) {
			label := "\033[33m" + prefix + "\033[0m"
			if prefix == "Error:" {
				label = "\033[31m" + prefix + "\033[0m"
			}
			return label + strings.TrimPrefix(msg, prefix)
		}
	}
	return msg
}

// printGomodResult renders a service action result: warnings to stderr
// (colored when TTY), KV block to stdout, errors as Error: on stderr.
func printGomodResult(ok bool, kv map[string]string, output, errMsg string) error {
	color := stderrColorEnabled()
	if output != "" {
		for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
			if strings.HasPrefix(line, "warning:") {
				if color {
					line = colorLabel(line)
				}
				fmt.Fprintln(os.Stderr, line)
				continue
			}
			fmt.Println(line)
		}
	}
	if len(kv) > 0 {
		printGomodKV(kv)
	}
	if !ok {
		if errMsg != "" {
			return fmt.Errorf("%s", errMsg)
		}
		return fmt.Errorf("action failed")
	}
	return nil
}

func printGomodKV(kv map[string]string) {
	order := []string{"status", "auto_start", "health", "port", "upstream", "root", "modules", "uptime", "last_ok", "misses"}
	width := 0
	seen := make(map[string]bool, len(kv))
	for _, k := range order {
		if _, ok := kv[k]; ok {
			seen[k] = true
			if len(k) > width {
				width = len(k)
			}
		}
	}
	for k := range kv {
		if !seen[k] {
			order = append(order, k)
			if len(k) > width {
				width = len(k)
			}
		}
	}
	pad := func(s string, w int) string {
		for len(s) < w {
			s += " "
		}
		return s
	}
	for _, k := range order {
		if v, ok := kv[k]; ok {
			fmt.Printf("  %s  %s\n", pad(k+":", width+1), v)
		}
	}
}
