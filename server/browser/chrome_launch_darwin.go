//go:build darwin

package browser

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type chromeProcess struct {
	PID      int
	DataDir  string
	Headless bool
}

// Darwin returns argc, executable, padding, argv and finally environment.
// Parse exactly argc entries so environment values are never exposed.
func parseChromeProcessArgs(raw []byte) ([]string, error) {
	if len(raw) < 4 {
		return nil, fmt.Errorf("short Chrome process arguments")
	}
	argc := int(binary.LittleEndian.Uint32(raw[:4]))
	if argc < 1 || argc > 4096 {
		return nil, fmt.Errorf("invalid Chrome argument count")
	}
	i := 4
	for i < len(raw) && raw[i] != 0 {
		i++
	}
	for i < len(raw) && raw[i] == 0 {
		i++
	}
	args := make([]string, 0, argc)
	for len(args) < argc {
		if i >= len(raw) {
			return nil, fmt.Errorf("incomplete Chrome process arguments")
		}
		j := i
		for j < len(raw) && raw[j] != 0 {
			j++
		}
		if j == len(raw) {
			return nil, fmt.Errorf("unterminated Chrome process argument")
		}
		args = append(args, string(raw[i:j]))
		i = j + 1
	}
	return args, nil
}

func chromeProcessFromArgs(pid int, args []string, defaultRoot string) (chromeProcess, error) {
	p := chromeProcess{PID: pid, DataDir: defaultRoot}
	if len(args) == 0 || !isChromeMainExecutable(args[0]) {
		return p, fmt.Errorf("profile lock owner is not a recognized Chrome browser process")
	}
	var dir string
	dirSet := false
	for i := 1; i < len(args); i++ {
		a := args[i]
		if a == "--headless" || strings.HasPrefix(a, "--headless=") {
			p.Headless = true
		}
		if strings.HasPrefix(a, "--user-data-dir=") {
			dirSet = true
			dir = strings.TrimPrefix(a, "--user-data-dir=")
		}
		if a == "--user-data-dir" {
			if i+1 >= len(args) {
				return p, fmt.Errorf("Chrome user-data argument is missing")
			}
			dirSet = true
			i++
			dir = args[i]
		}
	}
	if dirSet && dir == "" {
		return p, fmt.Errorf("Chrome user-data directory is empty")
	}
	if dirSet {
		p.DataDir = dir
	}
	if !filepath.IsAbs(p.DataDir) {
		return p, fmt.Errorf("Chrome user-data directory is not absolute")
	}
	resolved, err := filepath.EvalSymlinks(p.DataDir)
	if err != nil {
		if os.IsNotExist(err) {
			p.DataDir = filepath.Clean(p.DataDir)
			return p, nil
		}
		return p, fmt.Errorf("cannot verify Chrome user-data directory: %w", err)
	}
	p.DataDir = resolved
	return p, nil
}

func inspectChromePID(pid int, defaultRoot string) (chromeProcess, error) {
	raw, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return chromeProcess{}, fmt.Errorf("cannot inspect Chrome process %d: %w", pid, err)
	}
	args, err := parseChromeProcessArgs(raw)
	if err != nil {
		return chromeProcess{}, err
	}
	return chromeProcessFromArgs(pid, args, defaultRoot)
}

func isChromeMainExecutable(path string) bool {
	switch filepath.Base(path) {
	case "Google Chrome", "Google Chrome for Testing", "Chromium":
		return true
	default:
		return false
	}
}

// Inspect same-user browser processes by PID with native NUL-separated argv;
// never guess argument boundaries from ps's space-joined command column.
func verifyChromeProcesses(ctx context.Context, root, defaultRoot string) error {
	out, err := exec.CommandContext(ctx, "/bin/ps", "-axo", "pid=,uid=,comm=").Output()
	if err != nil {
		return fmt.Errorf("cannot list Chrome processes: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pid, e1 := strconv.Atoi(fields[0])
		uid, e2 := strconv.Atoi(fields[1])
		if e1 != nil || e2 != nil || uid != os.Getuid() || !isChromeMainExecutable(strings.Join(fields[2:], " ")) {
			continue
		}
		p, err := inspectChromePID(pid, defaultRoot)
		if err != nil && errors.Is(unix.Kill(pid, 0), unix.ESRCH) {
			continue
		}
		if err != nil {
			return err
		}
		if p.DataDir == root && p.Headless {
			return fmt.Errorf("the selected Chrome user-data directory is in use by headless Chrome (pid %d); stop or reconfigure that process before retrying", pid)
		}
	}
	return nil
}

// Chrome's singleton lock is authoritative for the user-data directory. Do
// not remove it, signal its owner, or attach to a profile of uncertain origin.
func verifyChromeProfileOwner(ctx context.Context, root, defaultRoot string) error {
	if err := verifyChromeProcesses(ctx, root, defaultRoot); err != nil {
		return err
	}
	lock, err := os.Readlink(filepath.Join(root, "SingletonLock"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot inspect Chrome profile lock: %w", err)
	}
	cut := strings.LastIndexByte(lock, '-')
	if cut < 0 {
		return fmt.Errorf("Chrome profile lock owner is unknown")
	}
	host, err := os.Hostname()
	if err != nil || lock[:cut] != host {
		return fmt.Errorf("Chrome profile lock belongs to an unknown host; check Chrome before retrying")
	}
	pid, err := strconv.Atoi(lock[cut+1:])
	if err != nil || pid <= 0 {
		return fmt.Errorf("Chrome profile lock owner is unknown")
	}
	owner, err := inspectChromePID(pid, defaultRoot)
	if err != nil && errors.Is(unix.Kill(pid, 0), unix.ESRCH) {
		return nil // Chrome itself may recover a stale lock.
	}
	if err != nil {
		return fmt.Errorf("cannot verify Chrome profile lock; do not remove it automatically: %w", err)
	}
	if owner.DataDir != root {
		return fmt.Errorf("Chrome profile lock does not match the owning process; check Chrome before retrying")
	}
	if owner.Headless {
		return fmt.Errorf("the selected Chrome user-data directory is in use by headless Chrome (pid %d); stop or reconfigure that process before retrying", pid)
	}
	return nil
}

func openChromeProfile(ctx context.Context, targetURL string, newWindow bool) error {
	return openChromeProfileWith(ctx, targetURL, newWindow, startChromeDetached)
}

// launch is injectable so routing tests own and observe only their temporary process.
func openChromeProfileWith(ctx context.Context, targetURL string, newWindow bool, launch func(string, []string) error) error {
	if !isExecutable(chromeBundledBinary) {
		return fmt.Errorf("Google Chrome was not found at %s", chromeBundledBinary)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("locate Chrome profile: %w", err)
	}
	defaultRoot := filepath.Join(home, "Library", "Application Support", "Google", "Chrome")
	root := strings.TrimSpace(os.Getenv("BROWSER_AGENT_CHROME_USER_DATA_DIR"))
	if root == "" {
		root = defaultRoot
	}
	profile, err := discoverChromeProfile(root, strings.TrimSpace(os.Getenv("BROWSER_AGENT_CHROME_PROFILE")))
	if err != nil {
		return err
	}
	if resolved, e := filepath.EvalSymlinks(defaultRoot); e == nil {
		defaultRoot = resolved
	}
	if err := verifyChromeProfileOwner(ctx, profile.Root, defaultRoot); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	args := []string{"--user-data-dir=" + profile.Root, "--profile-directory=" + profile.Name}
	if newWindow {
		args = append(args, "--new-window")
	}
	args = append(args, targetURL)
	return launch(chromeBundledBinary, args)
}

func startChromeDetached(binary string, args []string) error {
	// Do not tie Chrome's lifetime to the HTTP request.
	cmd := exec.Command(binary, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch Chrome in the selected profile: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
