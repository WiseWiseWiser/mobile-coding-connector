//go:build darwin

package browser

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func procArgsFixture(args ...string) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(len(args)))
	b = append(b, []byte(args[0])...)
	b = append(b, 0, 0, 0)
	for _, arg := range args {
		b = append(b, []byte(arg)...)
		b = append(b, 0)
	}
	return append(b, []byte("PRIVATE_ENV=not-an-argument\x00")...)
}

func TestChromeProcessArgumentsAndProfileRoot(t *testing.T) {
	root := t.TempDir()
	args := []string{chromeBundledBinary, "--headless=new", "--user-data-dir=" + root, "about:blank"}
	decoded, err := parseChromeProcessArgs(procArgsFixture(args...))
	if err != nil || len(decoded) != len(args) {
		t.Fatalf("decoded=%v err=%v", decoded, err)
	}
	for i := range args {
		if decoded[i] != args[i] {
			t.Fatalf("argv[%d]=%q", i, decoded[i])
		}
	}
	if bytes.Contains([]byte(strings.Join(decoded, " ")), []byte("PRIVATE_ENV")) {
		t.Fatal("environment included in args")
	}
	p, err := chromeProcessFromArgs(123, decoded, root)
	resolved, _ := filepath.EvalSymlinks(root)
	if err != nil || !p.Headless || p.DataDir != resolved {
		t.Fatalf("process=%+v err=%v", p, err)
	}
	if _, err := parseChromeProcessArgs([]byte{1, 0}); err == nil {
		t.Fatal("accepted short buffer")
	}
	if _, err := chromeProcessFromArgs(1, []string{"not-chrome", "--user-data-dir=" + root}, root); err == nil {
		t.Fatal("accepted non-Chrome owner")
	}
	if _, err := chromeProcessFromArgs(1, []string{chromeBundledBinary, "--user-data-dir="}, root); err == nil {
		t.Fatal("accepted empty user-data dir")
	}
}

func TestProfileDirectedLaunchUsesExplicitRootAndProfile(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	pkg := makeAgentPackage(t, root)
	writeProfile(t, root, "Profile 1", fmt.Sprintf(`{"a":{"path":%q}}`, pkg))
	t.Setenv("BROWSER_AGENT_CHROME_USER_DATA_DIR", root)
	t.Setenv("BROWSER_AGENT_CHROME_PROFILE", "Profile 1")
	var binary string
	var args []string
	launch := func(path string, a []string) error { binary = path; args = append([]string(nil), a...); return nil }
	if err := openChromeProfileWith(context.Background(), "http://127.0.0.1/test", true, launch); err != nil {
		t.Fatal(err)
	}
	if binary != chromeBundledBinary || len(args) != 4 || args[0] != "--user-data-dir="+root || args[1] != "--profile-directory=Profile 1" || args[2] != "--new-window" || args[3] != "http://127.0.0.1/test" {
		t.Fatalf("binary=%q args=%q", binary, args)
	}
}

func TestStaleLocalLockIsNotDeletedAndForeignLockIsRefused(t *testing.T) {
	root, _ := filepath.EvalSymlinks(t.TempDir())
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	defaultRoot := filepath.Join(home, "Library/Application Support/Google/Chrome")
	host, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	lock := filepath.Join(root, "SingletonLock")
	if err := os.Symlink(host+"-2147000000", lock); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := verifyChromeProfileOwner(ctx, root, defaultRoot); err != nil {
		t.Fatalf("stale local lock: %v", err)
	}
	if _, err := os.Lstat(lock); err != nil {
		t.Fatal("stale lock was modified")
	}
	other, _ := filepath.EvalSymlinks(t.TempDir())
	if err := os.Symlink("some-other-host-123", filepath.Join(other, "SingletonLock")); err != nil {
		t.Fatal(err)
	}
	if err := verifyChromeProfileOwner(ctx, other, defaultRoot); err == nil || !strings.Contains(err.Error(), "unknown host") {
		t.Fatalf("foreign lock error=%v", err)
	}
}

func TestChromeHeadlessProfileOwnerIsRejectedInIsolation(t *testing.T) {
	if os.Getenv("AI_CRITIC_TEST_ISOLATED_CHROME") != "1" {
		t.Skip("isolated Chrome test disabled")
	}
	root, _ := filepath.EvalSymlinks(t.TempDir())
	other, _ := filepath.EvalSymlinks(t.TempDir())
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	defaultRoot, err := filepath.EvalSymlinks(filepath.Join(home, "Library/Application Support/Google/Chrome"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(chromeBundledBinary, "--headless=new", "--no-first-run", "--user-data-dir="+root, "--remote-debugging-port=0", "about:blank")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }() // Only the child created by this test.
	deadline := time.Now().Add(20 * time.Second)
	for {
		if _, err := os.Lstat(filepath.Join(root, "SingletonLock")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("isolated Chrome did not acquire lock")
		}
		time.Sleep(100 * time.Millisecond)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := verifyChromeProfileOwner(ctx, root, defaultRoot); err == nil || !strings.Contains(err.Error(), "headless Chrome") {
		t.Fatalf("headless root error=%v", err)
	}
	if err := verifyChromeProfileOwner(ctx, other, defaultRoot); err != nil {
		t.Fatalf("unrelated isolated profile blocked: %v", err)
	}
	if err := syscall.Kill(cmd.Process.Pid, 0); err != nil {
		t.Fatalf("test process was unexpectedly stopped: %v", err)
	}
}

func TestCurrentChromeOwnerReadOnly(t *testing.T) {
	if os.Getenv("AI_CRITIC_TEST_CHROME_PROFILE") != "1" {
		t.Skip("host profile test disabled")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(filepath.Join(home, "Library/Application Support/Google/Chrome"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := verifyChromeProfileOwner(ctx, root, root); err != nil {
		t.Fatal(err)
	}
	t.Log("normal profile owner is verified; no process was changed")
}
