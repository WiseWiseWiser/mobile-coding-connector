// Smoke-test remote-agent ws-proxy vpn: start tunnel, curl google, stop.
//
// Usage:
//
//	go run ./script/wsproxy-vpn-smoke
//	go run ./script/wsproxy-vpn-smoke --attempts 3
//
// Hard limit: entire script exits within 2 minutes. On timeout or failure,
// VPN processes are killed and macOS DNS/proxy side effects are restored.
//
// Environment:
//
//	SUDO_PASSWORD  optional; passed to sudo -S (never commit passwords)
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
	readyMarker        = "TUN ready on"
	curlURL            = "https://www.google.com/generate_204"
	maxScriptDuration  = 2 * time.Minute
	settleAfterReady   = 5 * time.Second
	curlTimeout        = 12 * time.Second
	killGracePeriod    = 500 * time.Millisecond
)

func main() {
	attempts := 1
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			printUsage()
			return
		default:
			if strings.HasPrefix(arg, "--attempts") {
				var n int
				if _, err := fmt.Sscanf(arg, "--attempts=%d", &n); err == nil && n > 0 {
					attempts = n
				} else if _, err := fmt.Sscanf(strings.TrimPrefix(arg, "--attempts "), "%d", &n); err == nil && n > 0 {
					attempts = n
				}
			}
		}
	}

	scriptCtx, scriptCancel := context.WithTimeout(context.Background(), maxScriptDuration)
	defer scriptCancel()

	defer emergencyCleanup()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		select {
		case <-sigCh:
			emergencyCleanup()
			os.Exit(1)
		case <-scriptCtx.Done():
		}
	}()

	agent, err := resolveRemoteAgent()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for try := 1; try <= attempts; try++ {
		if err := scriptCtx.Err(); err != nil {
			fmt.Println("FAIL: script timed out after", maxScriptDuration)
			os.Exit(1)
		}
		if attempts > 1 {
			fmt.Printf("\n=== attempt %d/%d ===\n", try, attempts)
		}
		pass, detail := runVPNCurlSmoke(scriptCtx, agent)
		if pass {
			fmt.Println("PASS:", detail)
			os.Exit(0)
		}
		fmt.Println("FAIL:", detail)
	}
	os.Exit(1)
}

func printUsage() {
	fmt.Printf(`Usage: go run ./script/wsproxy-vpn-smoke [--attempts N]

Starts remote-agent ws-proxy vpn, waits for %q, runs:
  curl -4 -m 10 %s
then stops the VPN. Entire script is capped at %s.

On exit (pass, fail, timeout, or Ctrl+C), VPN processes are killed and
system DNS/proxy changes from the TUN session are restored.

Environment:
  SUDO_PASSWORD   optional; for non-interactive sudo -S
`, readyMarker, curlURL, maxScriptDuration)
}

func resolveRemoteAgent() (string, error) {
	if p, err := exec.LookPath("remote-agent"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("remote-agent not found in PATH")
}

func vpnCommand(agent string) *exec.Cmd {
	args := []string{"ws-proxy", "vpn"}
	if os.Geteuid() == 0 {
		return exec.Command(agent, args...)
	}
	sudoArgs := append([]string{"-S", agent}, args...)
	cmd := exec.Command("sudo", sudoArgs...)
	if pw := strings.TrimSpace(os.Getenv("SUDO_PASSWORD")); pw != "" {
		cmd.Stdin = strings.NewReader(pw + "\n")
	}
	return cmd
}

func runVPNCurlSmoke(ctx context.Context, agent string) (bool, string) {
	cmd := vpnCommand(agent)
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return false, fmt.Sprintf("stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return false, fmt.Sprintf("stderr pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return false, fmt.Sprintf("start vpn: %v", err)
	}
	stopped := false
	stopVPN := func() {
		if stopped {
			return
		}
		stopped = true
		stopVPNProcess(cmd)
	}
	defer func() {
		stopVPN()
		emergencyCleanup()
	}()

	ready := make(chan struct{})
	logDone := make(chan struct{})
	go streamUntilReady(ctx, io.MultiReader(stdout, stderr), ready, logDone)

	select {
	case <-ready:
	case <-ctx.Done():
		<-logDone
		return false, fmt.Sprintf("timed out before %q (%s)", readyMarker, ctx.Err())
	}

	select {
	case <-time.After(settleAfterReady):
	case <-ctx.Done():
		<-logDone
		return false, fmt.Sprintf("timed out before curl (%s)", ctx.Err())
	}

	curlOut, curlErr := runCurl(ctx)
	stopVPN()
	<-logDone

	if curlErr != nil {
		return false, fmt.Sprintf("curl: %v\n%s", curlErr, curlOut)
	}
	if strings.Contains(curlOut, "http_code=000") || strings.Contains(curlOut, "Connection reset") {
		return false, curlOut
	}
	if code := parseHTTPCode(curlOut); code != "" && code != "200" && code != "204" && code != "301" && code != "302" {
		return false, curlOut
	}
	return true, strings.TrimSpace(curlOut)
}

func streamUntilReady(ctx context.Context, r io.Reader, ready chan struct{}, done chan struct{}) {
	defer close(done)
	scanner := bufio.NewScanner(r)
	var readyOnce bool
	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		line := scanner.Text()
		fmt.Println(line)
		if !readyOnce && strings.Contains(line, readyMarker) {
			readyOnce = true
			close(ready)
		}
	}
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
		killOrphanVPNProcesses()
		if err := wsproxy_singbox.RestoreTunSessionSideEffects(); err != nil {
			fmt.Fprintf(os.Stderr, "restore tun side effects: %v\n", err)
		}
	})
}

func killOrphanVPNProcesses() {
	// Process group SIGINT first (handled by caller via onStop); then sweep orphans.
	for _, pattern := range []string{"sing-box run -c", "xray run -c", "remote-agent ws-proxy vpn"} {
		_ = exec.Command("pkill", "-f", pattern).Run()
	}
	time.Sleep(killGracePeriod)
	for _, pattern := range []string{"sing-box run -c", "xray run -c"} {
		_ = exec.Command("pkill", "-9", "-f", pattern).Run()
	}
}

func stopVPNProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	pgid := cmd.Process.Pid
	_ = syscall.Kill(-pgid, syscall.SIGINT)
	time.Sleep(killGracePeriod)
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
	_ = cmd.Wait()
}

func runCurl(ctx context.Context) (string, error) {
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < curlTimeout {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, remaining)
			defer cancel()
		}
	}
	args := []string{"-4", "-m", "10", "-sS", "-D", "-", "-o", "/dev/null", "-w", "http_code=%{http_code}\n", curlURL}
	cmd := exec.CommandContext(ctx, "curl", args...)
	cmd.Env = wsproxy_singbox.EnvWithoutProxy()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

