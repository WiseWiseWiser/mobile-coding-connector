# Scenario

**Feature**: remote-agent download directory integration harness

```
# serverHome + agentHome + agentWorkDir + session-cached binaries
leaf Setup -> seed serverHome and optional localDir -> remote-agent download -> local files + stdout
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` to scope a file cache under
   `$TMPDIR/remote-agent-download-dir-doctest-<session>/` (binaries built once per run).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaves.
3. Each leaf gets isolated `serverHome`, `agentHome`, and `agentWorkDir`; only compiled binaries are shared.
4. Server runs with `HOME=serverHome` and cwd `serverHome` so remote paths resolve there.
5. CLI runs with cwd `agentWorkDir` so relative local destinations land in an isolated work dir.

## Steps

1. Root `Run` builds binaries, creates `serverHome`/`agentHome`/`agentWorkDir`, applies `ServerPreseed*`
   and optional `LocalPreseed*`, starts `ai-critic-server` on an ephemeral port, writes agent config.
2. Leaf `Setup` seeds remote fixtures, sets `Request.Args`, `LocalDir`, and pre-seed maps.
3. `Run` executes `remote-agent --server ... --token ... download ...` with cwd `agentWorkDir`.
4. Leaf `Assert` checks exit code, CLI output, and files under `LocalDir` or `agentWorkDir`.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-download-dir.md and
REQUIREMENT-DESIGN-upload-download-dry-run.md. Directory downloads are
client-orchestrated fan-outs of per-file GET downloads with filesystem resume;
rejected leaves must show no partial local writes beyond pre-seeded state.
Dry-run leaves snapshot `localDir` before/after CLI (`snapshotDataTree` /
`assertTreeSnapshotUnchanged`).

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
	return filepath.Join(os.TempDir(), "remote-agent-download-dir-doctest-"+DOCTEST_SESSION_ID)
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

func setDownloadArgs(t *testing.T, req *Request, remotePath, localPath string) {
	t.Helper()
	setDownloadArgsWithDryRun(t, req, remotePath, localPath, false)
}

func setDownloadDryRunArgs(t *testing.T, req *Request, remotePath, localPath string) {
	t.Helper()
	setDownloadArgsWithDryRun(t, req, remotePath, localPath, true)
}

func setDownloadArgsWithDryRun(t *testing.T, req *Request, remotePath, localPath string, dryRun bool) {
	t.Helper()
	req.RemotePath = remotePath
	req.LocalPath = localPath
	args := []string{"download"}
	if dryRun {
		args = append(args, "--dry-run")
	}
	args = append(args, remotePath)
	if localPath != "" {
		args = append(args, localPath)
	}
	req.Args = args
}

func argsHasDryRun(args []string) bool {
	for _, a := range args {
		if a == "--dry-run" {
			return true
		}
	}
	return false
}

func snapshotDataTree(t *testing.T, root string) map[string]string {
	t.Helper()
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return map[string]string{}
	}
	out := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot data tree %s: %v", root, err)
	}
	return out
}

func assertTreeSnapshotUnchanged(t *testing.T, label string, before, after map[string]string) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("%s file count changed: before=%d after=%d", label, len(before), len(after))
	}
	for rel, want := range before {
		got, ok := after[rel]
		if !ok {
			t.Fatalf("%s missing file after CLI: %s", label, rel)
		}
		if got != want {
			t.Fatalf("%s file %q changed:\nwant len=%d\ngot len=%d", label, rel, len(want), len(got))
		}
	}
	for rel := range after {
		if _, ok := before[rel]; !ok {
			t.Fatalf("%s unexpected file after CLI: %s", label, rel)
		}
	}
}

func resolveLocalDir(agentWorkDir, remotePath, localPath string) string {
	base := filepath.Base(strings.TrimSuffix(filepath.ToSlash(remotePath), "/"))
	rel := localPath
	if rel == "" {
		rel = base
	} else if strings.HasSuffix(rel, "/") || strings.HasSuffix(rel, string(os.PathSeparator)) {
		rel = filepath.Join(strings.TrimSuffix(filepath.ToSlash(rel), "/"), base)
	}
	if filepath.IsAbs(rel) {
		return filepath.Clean(rel)
	}
	return filepath.Join(agentWorkDir, filepath.FromSlash(rel))
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

func applyLocalPreseed(t *testing.T, localDir string, files map[string]string, dirs []string) error {
	t.Helper()
	if localDir != "" {
		if err := os.MkdirAll(localDir, 0755); err != nil {
			return fmt.Errorf("mkdir localDir %s: %w", localDir, err)
		}
	}
	for _, rel := range dirs {
		full := filepath.Join(localDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(full, 0755); err != nil {
			return fmt.Errorf("mkdir local preseed dir %s: %w", full, err)
		}
	}
	for rel, content := range files {
		full := filepath.Join(localDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			return fmt.Errorf("mkdir local preseed parent %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			return fmt.Errorf("write local preseed file %s: %w", full, err)
		}
	}
	return nil
}

func localFilePath(root, rel string) string {
	return filepath.Join(root, filepath.FromSlash(rel))
}

func readLocalFile(t *testing.T, localDir, rel string) string {
	t.Helper()
	full := localFilePath(localDir, rel)
	data, err := os.ReadFile(full)
	if err != nil {
		t.Fatalf("read %s: %v", full, err)
	}
	return string(data)
}

func assertLocalFileContent(t *testing.T, localDir, rel, want string) {
	t.Helper()
	got := readLocalFile(t, localDir, rel)
	if got != want {
		t.Fatalf("local file %q content mismatch:\nwant: %q\ngot:  %q", rel, want, got)
	}
}

func assertLocalPathMissing(t *testing.T, localDir, rel string) {
	t.Helper()
	full := localFilePath(localDir, rel)
	if _, err := os.Stat(full); err == nil {
		t.Fatalf("expected missing local path %s", full)
	} else if !isLocalPathMissingErr(err) {
		t.Fatalf("stat %s: %v", full, err)
	}
}

func isLocalPathMissingErr(err error) bool {
	if os.IsNotExist(err) {
		return true
	}
	if errors.Is(err, syscall.ENOTDIR) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "not a directory")
}

func assertLocalIsDir(t *testing.T, localDir, rel string) {
	t.Helper()
	full := localFilePath(localDir, rel)
	info, err := os.Stat(full)
	if err != nil {
		t.Fatalf("stat %s: %v", full, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", full)
	}
}

func assertLocalDirEmpty(t *testing.T, localDir, rel string) {
	t.Helper()
	full := localFilePath(localDir, rel)
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

func repeatBytePattern(size int, seed byte) []byte {
	out := make([]byte, size)
	for i := range out {
		out[i] = byte((int(seed) + i) % 251)
	}
	return out
}
```