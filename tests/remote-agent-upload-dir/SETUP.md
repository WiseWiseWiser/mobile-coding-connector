# Scenario

**Feature**: remote-agent upload directory integration harness

```
# serverHome + agentHome + session-cached binaries
leaf Setup -> seed localDir and optional serverHome -> remote-agent upload -> remote files + stdout
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` to scope a file cache under
   `$TMPDIR/remote-agent-upload-dir-doctest-<session>/` (binaries built once per run).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaves.
3. Each leaf gets isolated `serverHome` and `agentHome`; only compiled binaries are shared.
4. Server runs with `HOME=serverHome` and cwd `serverHome` so remote paths resolve there.

## Steps

1. Root `Run` builds binaries, creates `serverHome`/`agentHome`, applies `ServerPreseed*`,
   starts `ai-critic-server` on an ephemeral port, writes agent config.
2. Leaf `Setup` creates local fixtures, sets `Request.Args`, `RemoteDir`, and pre-seed maps.
3. `Run` executes `remote-agent --server ... --token ... upload ...`.
4. Leaf `Assert` checks exit code, CLI output, and files under `serverHome`.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-upload-dir.md. Directory uploads are
client-orchestrated fan-outs of the existing per-file chunked upload API; rejected
leaves must show no partial writes under `remoteDir`.

```go
import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
)

func sessionCacheDir() string {
	return filepath.Join(os.TempDir(), "remote-agent-upload-dir-doctest-"+DOCTEST_SESSION_ID)
}

func withFileLock(t *testing.T, lockPath string, fn func() error) error {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock %s: %w", lockPath, err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return fn()
}

func buildSessionBinariesOnce(t *testing.T, moduleRoot, cacheDir string) (serverBin, agentBin string) {
	t.Helper()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	serverBin = filepath.Join(cacheDir, "ai-critic-server")
	agentBin = filepath.Join(cacheDir, "remote-agent")
	ready := filepath.Join(cacheDir, "binaries.ready")
	lock := filepath.Join(cacheDir, "build.lock")
	err := withFileLock(t, lock, func() error {
		if fileExists(ready) && fileExists(serverBin) && fileExists(agentBin) {
			return nil
		}
		for _, spec := range []struct {
			out string
			pkg string
		}{
			{serverBin, "."},
			{agentBin, "./cmd/remote-agent"},
		} {
			cmd := exec.Command("go", "build", "-o", spec.out, spec.pkg)
			cmd.Dir = moduleRoot
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("build %s: %w\n%s", spec.pkg, err, string(out))
			}
		}
		return os.WriteFile(ready, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("session binaries cache: %s", cacheDir)
	return serverBin, agentBin
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func Setup(t *testing.T, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	return nil
}

func mkLocalWorkDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "upload-dir-local-*")
	if err != nil {
		t.Fatalf("mkdir local work dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func writeLocalFile(t *testing.T, root, rel, content string, perm os.FileMode) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if perm == 0 {
		perm = 0644
	}
	if err := os.WriteFile(full, []byte(content), perm); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func writeLocalDir(t *testing.T, root, rel string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(full, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", full, err)
	}
}

func setUploadArgs(t *testing.T, req *Request, localPath, remotePath string) {
	t.Helper()
	abs, err := filepath.Abs(localPath)
	if err != nil {
		t.Fatalf("abs local path: %v", err)
	}
	req.LocalPath = abs
	req.RemotePath = remotePath
	args := []string{"upload", abs}
	if remotePath != "" {
		args = append(args, remotePath)
	}
	req.Args = args
}

func resolveRemoteDir(t *testing.T, serverHome, localPath, remotePath string) string {
	t.Helper()
	base := filepath.Base(localPath)
	rel := remotePath
	if rel == "" {
		rel = base + "/"
	} else if strings.HasSuffix(rel, "/") {
		rel = rel + base
	}
	if strings.HasPrefix(rel, "/") {
		return filepath.Clean(rel)
	}
	return filepath.Join(serverHome, filepath.FromSlash(rel))
}

func applyServerPreseed(t *testing.T, serverHome string, files map[string]string, dirs []string) error {
	t.Helper()
	for _, rel := range dirs {
		full := filepath.Join(serverHome, filepath.FromSlash(rel))
		if err := os.MkdirAll(full, 0755); err != nil {
			return fmt.Errorf("mkdir preseed dir %s: %w", full, err)
		}
	}
	for rel, content := range files {
		full := filepath.Join(serverHome, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return fmt.Errorf("mkdir preseed parent %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			return fmt.Errorf("write preseed file %s: %w", full, err)
		}
	}
	return nil
}

func serverFilePath(serverHome, rel string) string {
	return filepath.Join(serverHome, filepath.FromSlash(rel))
}

func readServerFile(t *testing.T, serverHome, rel string) string {
	t.Helper()
	full := serverFilePath(serverHome, rel)
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read %s: %v", full, err)
	}
	return string(data)
}

func assertServerFileContent(t *testing.T, serverHome, rel, want string) {
	t.Helper()
	got := readServerFile(t, serverHome, rel)
	if got != want {
		t.Fatalf("server file %q content mismatch:\nwant: %q\ngot:  %q", rel, want, got)
	}
}

func assertServerPathMissing(t *testing.T, serverHome, rel string) {
	t.Helper()
	full := serverFilePath(serverHome, rel)
	if _, err := os.Stat(full); err == nil {
		t.Fatalf("expected missing server path %s", full)
	} else if !isServerPathMissingErr(err) {
		t.Fatalf("stat %s: %v", full, err)
	}
}

func isServerPathMissingErr(err error) bool {
	if os.IsNotExist(err) {
		return true
	}
	if errors.Is(err, syscall.ENOTDIR) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "not a directory")
}

func assertServerIsDir(t *testing.T, serverHome, rel string) {
	t.Helper()
	full := serverFilePath(serverHome, rel)
	info, err := os.Stat(full)
	if err != nil {
		t.Fatalf("stat %s: %v", full, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", full)
	}
}

func assertServerDirEmpty(t *testing.T, serverHome, rel string) {
	t.Helper()
	full := serverFilePath(serverHome, rel)
	entries, err := os.ReadDir(full)
	if err != nil {
		t.Fatalf("readdir %s: %v", full, err)
	}
	if len(entries) != 0 {
		t.Fatalf("%s expected empty, got %d entries", full, len(entries))
	}
}

func assertStdoutEndsWithNewline(t *testing.T, stdout string) {
	t.Helper()
	if stdout == "" {
		t.Fatal("stdout empty; want trailing newline")
	}
	if !strings.HasSuffix(stdout, "\n") {
		t.Fatalf("stdout missing trailing newline; ends with %q", stdout[len(stdout)-min(40, len(stdout)):])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func combinedHasAll(t *testing.T, combined string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(combined, n) {
			t.Fatalf("output missing %q;\nhave:\n%s", n, combined)
		}
	}
}

func combinedHasNone(t *testing.T, combined string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if strings.Contains(combined, n) {
			t.Fatalf("output unexpectedly contains %q;\nhave:\n%s", n, combined)
		}
	}
}

func copyFixture(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open fixture %s: %v", src, err)
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(dst), err)
	}
	out, err := os.Create(dst)
	if err != nil {
		t.Fatalf("create %s: %v", dst, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("copy %s -> %s: %v", src, dst, err)
	}
}
```