// Client-side smoke helper for remote-agent ws-proxy vpn-http-only.
//
// Assumes ws-proxy is already running on the configured remote server.
// Prompts you to start vpn-http-only in another terminal (sudo), then runs curl checks.
//
// Usage:
//
//	go run ./script/wsproxy-vpn-http-only-smoke
//	go run ./script/wsproxy-vpn-http-only-smoke --try-url https://internal.example.com
//	go run ./script/wsproxy-vpn-http-only-smoke --whitelist --include '*.corp.internal'
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/xhd2015/ai-critic/cmd/remote-agent/wsproxy_singbox"
)

var globalCleanupOnce sync.Once

const (
	curlURL           = "https://www.google.com/generate_204"
	maxScriptDuration = 10 * time.Minute
	curlTimeout       = 12 * time.Second
)

type options struct {
	tryURL      string
	skipDoctor  bool
	skipStatus  bool
	whitelist   bool
	blacklist   bool
	dnsHijack   bool
	includes    []string
	excludes    []string
}

func main() {
	opts, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	scriptCtx, scriptCancel := context.WithTimeout(context.Background(), maxScriptDuration)
	defer scriptCancel()

	defer emergencyCleanup()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		emergencyCleanup()
		os.Exit(1)
	}()

	agent, err := resolveRemoteAgent()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := runSmoke(scriptCtx, agent, opts); err != nil {
		fmt.Println("FAIL:", err)
		os.Exit(1)
	}
	fmt.Println("PASS: vpn-http-only smoke checks completed")
}

func parseArgs(args []string) (options, error) {
	opts := options{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		case "--try-url":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--try-url requires a value")
			}
			i++
			opts.tryURL = args[i]
		case "--skip-doctor":
			opts.skipDoctor = true
		case "--skip-status":
			opts.skipStatus = true
		case "--whitelist":
			opts.whitelist = true
		case "--blacklist":
			opts.blacklist = true
		case "--dns-hijack":
			opts.dnsHijack = true
		case "--include":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--include requires a value")
			}
			i++
			opts.includes = append(opts.includes, args[i])
		case "--exclude":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--exclude requires a value")
			}
			i++
			opts.excludes = append(opts.excludes, args[i])
		default:
			return opts, fmt.Errorf("unknown argument %q", arg)
		}
	}
	if opts.whitelist && opts.blacklist {
		return opts, fmt.Errorf("--whitelist and --blacklist are mutually exclusive")
	}
	return opts, nil
}

func printUsage() {
	fmt.Print(`Usage: go run ./script/wsproxy-vpn-http-only-smoke [options]

Client-side smoke helper. Requires ws-proxy already running on the remote server.

Options:
  --try-url URL        Additional HTTPS URL to curl (no HTTP_PROXY env)
  --whitelist          Forwarded to suggested vpn-http-only command
  --blacklist          Forwarded to suggested vpn-http-only command
  --include PATTERN    Repeatable
  --exclude PATTERN    Repeatable
  --dns-hijack         Forwarded to suggested vpn-http-only command
  --skip-doctor        Skip ws-proxy doctor preflight
  --skip-status        Skip ws-proxy status preflight

The script prints a sudo command for another terminal, waits for Enter after
"TUN ready on", runs curl checks, then waits for you to stop vpn-http-only.
`)
}

func resolveRemoteAgent() (string, error) {
	if p, err := exec.LookPath("remote-agent"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("remote-agent not found in PATH")
}

func runSmoke(ctx context.Context, agent string, opts options) error {
	if !opts.skipStatus {
		fmt.Println("=== [1/5] ws-proxy status ===")
		if err := runAgentStep(agent, "ws-proxy", "status"); err != nil {
			return fmt.Errorf("ws-proxy status: %w (start ws-proxy on remote first)", err)
		}
	}

	if !opts.skipDoctor {
		fmt.Println("=== [2/5] ws-proxy doctor ===")
		if err := runAgentStep(agent, "ws-proxy", "doctor"); err != nil {
			fmt.Println("warning: doctor reported issues; continuing smoke test")
		}
	} else {
		fmt.Println("=== [2/5] ws-proxy doctor (skipped) ===")
	}

	fmt.Println("=== [2b/5] DNS pollution check ===")
	wsproxy_singbox.MaybeWarnDNSPollution(opts.dnsHijack)

	fmt.Println("=== [3/5] start vpn-http-only (manual) ===")
	fmt.Println("Run this in another terminal and enter your sudo password if prompted:")
	fmt.Println()
	fmt.Printf("  %s\n", suggestedVPNCommand(agent, opts))
	fmt.Println()
	fmt.Println("Wait until you see \"TUN ready on utun\", then return here.")
	if !waitForEnter("Press Enter to continue...") {
		return ctx.Err()
	}

	fmt.Println("=== [4/5] curl checks (no HTTP_PROXY) ===")
	out, err := runCurlURL(ctx, curlURL)
	if err != nil {
		return fmt.Errorf("curl %s: %w\n%s", curlURL, err, out)
	}
	fmt.Println("OK:", strings.TrimSpace(outFirstLine(out)))

	if opts.tryURL != "" {
		if out, err := runCurlURL(ctx, opts.tryURL); err != nil {
			return fmt.Errorf("curl %s: %w\n%s", opts.tryURL, err, out)
		}
		fmt.Printf("OK: --try-url %s\n", opts.tryURL)
	}

	fmt.Println("=== [5/5] stop vpn-http-only (manual) ===")
	fmt.Println("Press Ctrl+C in the vpn-http-only terminal (or stop sing-box), then continue.")
	if !waitForEnter("Press Enter after vpn-http-only has stopped...") {
		return ctx.Err()
	}
	emergencyCleanup()
	return nil
}

func suggestedVPNCommand(agent string, opts options) string {
	var parts []string
	if os.Geteuid() != 0 {
		parts = append(parts, "sudo")
	}
	parts = append(parts, shellQuote(agent), "ws-proxy", "vpn-http-only")
	if opts.dnsHijack {
		parts = append(parts, "--dns-hijack")
	}
	if opts.whitelist {
		parts = append(parts, "--whitelist")
	}
	if opts.blacklist {
		parts = append(parts, "--blacklist")
	}
	for _, inc := range opts.includes {
		parts = append(parts, "--include", shellQuote(inc))
	}
	for _, exc := range opts.excludes {
		parts = append(parts, "--exclude", shellQuote(exc))
	}
	return strings.Join(parts, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '/' && r != '_' && r != '-' && r != '.' && r != ':' && r != '*' {
			return fmt.Sprintf("%q", s)
		}
	}
	if strings.Contains(s, "*") {
		return fmt.Sprintf("%q", s)
	}
	return s
}

func runAgentStep(agent string, args ...string) error {
	cmd := exec.Command(agent, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}

func waitForEnter(prompt string) bool {
	fmt.Print(prompt + " ")
	reader := bufio.NewReader(os.Stdin)
	_, err := reader.ReadString('\n')
	return err == nil
}

func outFirstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func runCurlURL(ctx context.Context, url string) (string, error) {
	args := []string{"-4", "-m", "10", "-sS", "-D", "-", "-o", "/dev/null", "-w", "http_code=%{http_code}\n", url}
	cmd := exec.CommandContext(ctx, "curl", args...)
	cmd.Env = wsproxy_singbox.EnvWithoutProxy()
	out, err := cmd.CombinedOutput()
	text := string(out)
	if err != nil {
		return text, err
	}
	if strings.Contains(text, "http_code=000") {
		return text, fmt.Errorf("connection failed")
	}
	code := parseHTTPCode(text)
	if code != "" && code != "200" && code != "204" && code != "301" && code != "302" {
		return text, fmt.Errorf("HTTP %s", code)
	}
	return text, nil
}

func parseHTTPCode(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "http_code=") {
			return strings.TrimPrefix(line, "http_code=")
		}
	}
	return ""
}

func emergencyCleanup() {
	globalCleanupOnce.Do(func() {
		for _, pattern := range []string{
			"sing-box run -c",
			"xray run -c",
			"remote-agent ws-proxy vpn-http-only",
			"remote-agent ws-proxy vpn",
		} {
			_ = exec.Command("pkill", "-f", pattern).Run()
		}
		time.Sleep(500 * time.Millisecond)
		if err := wsproxy_singbox.RestoreTunSessionSideEffects(); err != nil {
			fmt.Fprintf(os.Stderr, "restore tun side effects: %v\n", err)
		}
	})
}

