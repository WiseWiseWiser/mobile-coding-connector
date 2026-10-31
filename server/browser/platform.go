//go:build darwin || linux

package browser

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/xhd2015/ai-critic/server/config"
)

// findBinary locates the browser-agent CLI. An explicit override wins so a
// development build can be used without touching PATH.
func findBinary() string {
	if p := strings.TrimSpace(os.Getenv("BROWSER_AGENT_BIN")); p != "" {
		if isExecutable(p) {
			return p
		}
	}
	if p, err := exec.LookPath("browser-agent"); err == nil {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, rel := range []string{".local/bin/browser-agent", "go/bin/browser-agent"} {
		p := filepath.Join(home, rel)
		if isExecutable(p) {
			return p
		}
	}
	return ""
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0111 != 0
}

// spawnDaemon starts `browser-agent serve` detached so it outlives ai-critic and
// its restarts. The daemon is shared state: the extension talks to that port.
func (m *Manager) spawnDaemon() error {
	logPath := config.BrowserDaemonLogFile
	if dir := filepath.Dir(logPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	cmd := exec.Command(m.binary, "serve", "--port", strconv.Itoa(m.port))
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	// A new session detaches the daemon from ai-critic's process group, so a
	// crash-restart of the server does not kill the browser it was driving.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		logFile.Close()
		return err
	}
	go func() {
		_ = cmd.Wait()
		logFile.Close()
	}()
	return nil
}

// openInChrome opens url in the user's real Chrome profile: that is the whole
// point of this feature, since the extension runs there with the user's logins.
func openInChrome(ctx context.Context, url string) error {
	// Automation opens the session page in its own isolated Chrome, so it must
	// not also pop a window in the user's real profile.
	if os.Getenv("BROWSER_AGENT_NO_OPEN") == "1" {
		return nil
	}
	return openChromeProfile(ctx, url, false)
}

// chromeBundledBinary is where Chrome lives on macOS; Linux distributions put
// it on PATH under one of the usual names.
var chromeBundledBinary = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

// openNewChromeWindow opens url in a brand-new Chrome window. The extension has
// no window-creation job, so this is a host-level launch: Chrome's
// --new-window flag creates the window and the URL loads in it, which is what
// makes a second session have its own window (and therefore its own tab set).
func openNewChromeWindow(ctx context.Context, url string) error {
	if os.Getenv("BROWSER_AGENT_NO_OPEN") == "1" {
		return nil
	}
	return openChromeProfile(ctx, url, true)
}
