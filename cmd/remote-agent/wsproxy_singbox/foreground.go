package wsproxy_singbox

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
)

const (
	singBoxReadyTimeout  = 20 * time.Second
	singBoxLogPollPeriod = 150 * time.Millisecond
)

func runSingBoxForeground(ctx context.Context, sudo bool, configPath string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logPath := singBoxLogPath()

	PrintCommand(SingBoxRunCommand(sudo, configPath))
	fmt.Println("sing-box runs in foreground until you press Ctrl+C.")
	fmt.Printf("Log file: %s\n", logPath)
	if sudo {
		fmt.Println("(enter sudo password if prompted)")
	}

	logOffset := int64(0)
	if fi, err := os.Stat(logPath); err == nil {
		logOffset = fi.Size()
	}

	networkService, _ := activeNetworkService()
	var previousProxy serviceProxyState
	var proxyTouched bool
	if networkService != "" {
		if state, err := getServiceProxyState(networkService); err == nil {
			previousProxy = state
		}
	}
	var previousDNS []string
	if networkService != "" {
		previousDNS, _ = getDNSServers(networkService)
	}

	restoreProxy, proxyErr := disableSystemProxiesForTun()
	if proxyErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not disable system proxy: %v\n", proxyErr)
		restoreProxy = nil
	} else {
		proxyTouched = previousProxy.web.enabled || previousProxy.secure.enabled || previousProxy.socks.enabled
	}
	var restoreSession sync.Once
	runRestoreSession := func() {
		restoreSession.Do(func() {
			if restoreProxy != nil {
				restoreProxy()
			}
			clearTunSessionSnapshot()
		})
	}
	defer runRestoreSession()

	args := []string{"sing-box", "run", "-c", configPath}
	if sudo {
		args = append([]string{"sudo"}, args...)
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = singBoxProcessEnv()
	if err := cmd.Start(); err != nil {
		return err
	}

	fmt.Printf("sing-box process started (PID %d), waiting for TUN...\n", cmd.Process.Pid)

	logOffset, tunIface, err := waitForSingBoxReady(ctx, logPath, logOffset, singBoxReadyTimeout)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}

	if verifyLocalProxyAfterTun != nil {
		if err := verifyLocalProxyAfterTun(); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			runRestoreSession()
			return fmt.Errorf("xray lost VMess path after TUN routes came up: %w (see %s)", err, xraySidecarLogPath())
		}
		fmt.Println("xray VMess path verified with TUN routes active.")
	}

	var restoreDNS func()
	var dnsTouched bool
	if shouldConfigurePlatformTunDNS() {
		var dnsErr error
		restoreDNS, dnsErr = configurePlatformTunDNS()
		if dnsErr != nil {
			fmt.Fprintf(os.Stderr, "warning: could not set system DNS to %s: %v\n", tunDNSAddress, dnsErr)
			restoreDNS = nil
		}
		dnsTouched = restoreDNS != nil
	}
	if networkService != "" && (dnsTouched || proxyTouched) {
		if err := saveTunSessionSnapshot(networkService, previousDNS, dnsTouched, previousProxy, proxyTouched); err != nil {
			fmt.Fprintf(os.Stderr, "warning: save tun session snapshot: %v\n", err)
		}
	}
	var restoreOnce sync.Once
	runRestoreDNS := func() {
		if restoreDNS == nil {
			return
		}
		restoreOnce.Do(func() {
			restoreDNS()
			clearTunSessionSnapshot()
		})
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		select {
		case <-sigCh:
			runRestoreDNS()
			runRestoreSession()
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			cancel()
		case <-ctx.Done():
		}
	}()

	if tunIface != "" {
		fmt.Printf("TUN ready on %s. Streaming logs (Ctrl+C to stop):\n", tunIface)
	} else {
		fmt.Println("sing-box started. Streaming logs (Ctrl+C to stop):")
	}
	fmt.Println("Keep this terminal open. Test in another tab:")
	fmt.Println("  curl -4 -m 10 https://www.google.com")
	fmt.Println(strings.Repeat("-", 60))

	errCh := make(chan error, 1)
	go func() {
		errCh <- streamSingBoxLogFollow(ctx, logPath, logOffset)
	}()

	waitErr := cmd.Wait()
	streamErr := <-errCh
	runRestoreDNS()
	runRestoreSession()
	if waitErr != nil {
		return waitErr
	}
	return streamErr
}

func waitForSingBoxReady(ctx context.Context, logPath string, offset int64, timeout time.Duration) (readyOffset int64, tunIface string, err error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return 0, "", err
		}

		var ready bool
		offset, ready, tunIface, err = readSingBoxLogSince(logPath, offset, true, func(line string) {
			fmt.Println(line)
		})
		if err != nil && !os.IsNotExist(err) {
			return 0, "", fmt.Errorf("read sing-box log: %w", err)
		}
		if ready {
			return offset, tunIface, nil
		}

		select {
		case <-ctx.Done():
			return 0, "", ctx.Err()
		case <-time.After(singBoxLogPollPeriod):
		}
	}

	return 0, "", fmt.Errorf("sing-box did not become ready within %s; check %s", timeout, logPath)
}

func streamSingBoxLogFollow(ctx context.Context, logPath string, offset int64) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		var err error
		offset, _, _, err = readSingBoxLogSince(logPath, offset, false, func(line string) {
			fmt.Println(line)
		})
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read sing-box log: %w", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(singBoxLogPollPeriod):
		}
	}
}

func readSingBoxLogSince(path string, offset int64, failOnError bool, onLine func(string)) (newOffset int64, ready bool, tunIface string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return offset, false, "", err
	}
	defer f.Close()

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return offset, false, "", err
		}
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		onLine(line)
		if iface, ok := parseTunStartedLine(line); ok {
			ready = true
			tunIface = iface
		}
		if strings.Contains(line, "sing-box started") {
			ready = true
		}
		if failOnError && !ready && (strings.Contains(line, "FATAL") || strings.Contains(line, "ERROR")) {
			return currentFileOffset(f), false, "", fmt.Errorf("%s", line)
		}
	}
	if err := scanner.Err(); err != nil {
		return offset, ready, tunIface, err
	}

	return currentFileOffset(f), ready, tunIface, nil
}

func currentFileOffset(f *os.File) int64 {
	pos, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0
	}
	return pos
}

func parseTunStartedLine(line string) (string, bool) {
	const marker = "inbound/tun"
	if !strings.Contains(line, marker) || !strings.Contains(line, "started at") {
		return "", false
	}
	idx := strings.LastIndex(line, "started at ")
	if idx < 0 {
		return "", true
	}
	iface := strings.TrimSpace(line[idx+len("started at "):])
	return iface, true
}