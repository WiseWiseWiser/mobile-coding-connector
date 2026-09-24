# Scenario

**Feature**: remote-agent edit doctest harness (L2 mass + one L3 smoke)

```
# remote file -> staged copy under --work-dir -> local editor -> md5-guarded write-back
serverHome file -> remote-agent edit <file> -> staged file + editor + POST /api/files/write
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` to scope a file cache under
   `$TMPDIR/remote-agent-edit-doctest-<session>/` (L3 binaries when needed).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaves.
3. Each leaf gets isolated `serverHome`, `agentHome`, and `agentWorkDir`; only compiled binaries are shared.
4. **L2** (default): in-process mux with `fileupload.RegisterAPIForHome(mux, serverHome)`
   + `agentcli.Run`; stdin is swapped to `/dev/null` so terminal-editor guards are
   deterministic, and `HOME` is pointed at `agentHome` for config isolation.
5. **L3** (`UseCLI` smoke only): product binaries (`ai-critic-server` + `remote-agent`)
   with `HOME=agentHome` for the CLI.

## Steps

1. Root `Run` (see DOCTEST.md) seeds homes; L2 starts the in-process API, L3 builds/starts product binaries.
2. Leaf `Setup` seeds remote fixtures and sets `Request.Args`, `RemoteRel`, and editor knobs.
3. `Run` writes the fake editor script, injects `--work-dir <agentWorkDir>/staging`
   (unless the args already carry one), and executes `edit`.
4. Leaf `Assert` checks exit code, CLI output, staged copy, and remote file state.

## Context

Implements the `remote-agent edit` design: staging under
`/tmp/remote-agent-edit/<remote-path>` (overridable with `--work-dir`), editor
selection via `--editor`/`$VISUAL`/`$EDITOR`, `file not changed` short-circuit,
and `POST /api/files/write` with the download-time md5 as precondition (409 keeps
the staged copy). Leaves assert remote bytes/mode and staged bytes rather than
hashes, so the tests stay readable.

```go
import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/ai-critic/script/lib"
	"github.com/xhd2015/doctest/session"
)

func sessionCacheDir(sessionID string) string {
	return filepath.Join(os.TempDir(), "remote-agent-edit-doctest-"+sessionID)
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

// standardRemoteFile is the fixture used by most leaves.
const standardRemoteFile = "notes.md"
const standardRemoteInitial = "old\n"

// setEditArgs seeds a remote file and points the CLI at it by relative path.
func setEditArgs(req *Request, remoteRel, content string) {
	req.FileArg = remoteRel
	req.RemoteRel = remoteRel
	req.ServerPreseedFiles = map[string]string{remoteRel: content}
	req.Args = []string{"edit", remoteRel}
}

// setEditNewFileArgs points the CLI at a remote path that does not exist yet.
func setEditNewFileArgs(req *Request, remoteRel string) {
	req.FileArg = remoteRel
	req.RemoteRel = remoteRel
	req.Args = []string{"edit", remoteRel}
}

// setConflictArgs seeds a remote file and asks the editor to write both sides.
func setConflictArgs(req *Request, remoteRel, initial, staged, remote string) {
	req.FileArg = remoteRel
	req.RemoteRel = remoteRel
	req.ServerPreseedFiles = map[string]string{remoteRel: initial}
	req.Args = []string{"edit", remoteRel}
	req.EditorWrite = staged
	req.RemoteWrite = map[string]string{remoteRel: remote}
}

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	if d.DOCTEST_SESSION_ID == "" {
		t.Fatal("session id empty on session.Doctest")
	}
	return nil
}

// --- assertions shared by leaves ---

func assertExit(t *testing.T, resp *Response, want int) {
	t.Helper()
	if resp.ExitCode != want {
		t.Fatalf("exit = %d, want %d;\nstdout:\n%s\nstderr:\n%s", resp.ExitCode, want, resp.Stdout, resp.Stderr)
	}
}

func assertSecondExit(t *testing.T, resp *Response, want int) {
	t.Helper()
	if resp.SecondExitCode != want {
		t.Fatalf("second exit = %d, want %d;\nstdout:\n%s\nstderr:\n%s",
			resp.SecondExitCode, want, resp.SecondStdout, resp.SecondStderr)
	}
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

func assertRemoteContent(t *testing.T, resp *Response, rel, want string) {
	t.Helper()
	path := filepath.Join(resp.ServerHome, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read remote %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("remote %s content mismatch:\nwant: %q\ngot:  %q", rel, want, string(data))
	}
}

func assertRemoteMissing(t *testing.T, resp *Response, rel string) {
	t.Helper()
	path := filepath.Join(resp.ServerHome, filepath.FromSlash(rel))
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("remote %s should be missing, stat err = %v", rel, err)
	}
}

func assertRemoteMode(t *testing.T, resp *Response, rel string, want os.FileMode) {
	t.Helper()
	path := filepath.Join(resp.ServerHome, filepath.FromSlash(rel))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat remote %s: %v", rel, err)
	}
	if info.Mode().Perm() != want {
		t.Fatalf("remote %s mode = %v, want %v", rel, info.Mode().Perm(), want)
	}
}

func assertRemoteIsSymlink(t *testing.T, resp *Response, rel string) {
	t.Helper()
	path := filepath.Join(resp.ServerHome, filepath.FromSlash(rel))
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat remote %s: %v", rel, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("remote %s is not a symlink anymore (mode %v)", rel, info.Mode())
	}
}

func assertStagedContent(t *testing.T, resp *Response, want string) {
	t.Helper()
	data, err := os.ReadFile(resp.StagedPath)
	if err != nil {
		t.Fatalf("read staged %s: %v", resp.StagedPath, err)
	}
	if string(data) != want {
		t.Fatalf("staged content mismatch:\nwant: %q\ngot:  %q", want, string(data))
	}
}

func assertStagedMode(t *testing.T, resp *Response, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(resp.StagedPath)
	if err != nil {
		t.Fatalf("stat staged %s: %v", resp.StagedPath, err)
	}
	if info.Mode().Perm() != want {
		t.Fatalf("staged mode = %v, want %v", info.Mode().Perm(), want)
	}
}

func assertStagedPathUnderStaging(t *testing.T, resp *Response) {
	t.Helper()
	if !strings.HasPrefix(resp.StagedPath, resp.StagingDir+string(os.PathSeparator)) {
		t.Fatalf("staged path %s is not under staging dir %s", resp.StagedPath, resp.StagingDir)
	}
	if resp.StagedPath != filepath.Join(resp.StagingDir, resp.RemotePath) {
		t.Fatalf("staged path = %s, want %s", resp.StagedPath, filepath.Join(resp.StagingDir, resp.RemotePath))
	}
}

func md5Hex(content string) string {
	sum := md5.Sum([]byte(content))
	return hex.EncodeToString(sum[:])
}

func assertEditorDidNotRun(t *testing.T, resp *Response) {
	t.Helper()
	if resp.EditorRan {
		t.Fatal("editor executable ran, but the run should have failed before launching it")
	}
}

// requestCount counts the L2 server requests for an exact path.
func requestCount(resp *Response, path string) int {
	n := 0
	for _, p := range resp.Requests {
		if p == path {
			n++
		}
	}
	return n
}

// assertRequestCount pins how many times a path was requested, which is how the
// download-skip optimization is proven (stdout alone cannot show it).
func assertRequestCount(t *testing.T, resp *Response, path string, want int) {
	t.Helper()
	if got := requestCount(resp, path); got != want {
		t.Fatalf("requests to %s = %d, want %d;\nall requests:\n%s",
			path, got, want, strings.Join(resp.Requests, "\n"))
	}
}
```
