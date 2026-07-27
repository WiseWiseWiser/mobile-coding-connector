# Scenario

**Feature**: remote-agent paste-bin doctest harness (L2 mass + L3 smokes)

```
# L2: RegisterAPIForDir + agentcli.Run | L3 UseCLI: product binaries
leaf Setup -> seed/reset scratch -> paste-bin -> stdout/stderr + API scratch
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` to scope a file cache under
   `$TMPDIR/remote-agent-paste-bin-doctest-<session>/` (L3 binaries when needed).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaves.
3. Each leaf gets isolated `configHome` and `agentHome`; only compiled binaries are shared.
4. **L2** (default): in-process mux with `filetransfer.RegisterAPIForDir(mux, configHome/file-transfer)`
   + local bearer auth + `agentcli.Run` with mutex-scoped stdin swap (no process Setenv).
5. **L3** (`UseCLI` smokes only): product binaries with `AI_CRITIC_HOME=configHome`.
6. Read leaves use TTY stdin (no `PipedStdin` → L2 uses `/dev/null`). Write leaves set `PipedStdin` and `StdinBytes`.

## Steps

1. Root `Run` creates `configHome`/`agentHome`, applies scratch seed/reset; L2 starts in-process API, L3 builds/starts product binaries.
2. Leaf `Setup` sets `Request.Args`, scratch state, and stdin pipe mode; smokes set `UseCLI`.
3. `Run` executes `paste-bin` via agentcli (L2) or product binary (L3).
4. `Run` fetches scratch via `GET /api/file-transfer/scratch` after CLI for side-effect checks.
5. Leaf `Assert` checks exit code, CLI output, and scratch API state.

## Context

Implements REQUIREMENT-DESIGN-remote-agent-paste-bin.md. Classic TDD: command and client
methods are absent until implementation; all leaves are RED initially.

```go
import (
	"encoding/base64"
	"encoding/json"
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

const (
	scratchB64Prefix = "paste-bin:b64:"

	seededUTF8Content   = "line1\nline2\nemoji🎉"
	seededMetaUpdatedAt = "2026-07-14T08:30:00Z"
	staleScratchContent = "stale-scratch-content"
	forceReadSeedContent = "seeded-scratch-for-force-read"
	forceReadIgnoredPipe = "junk-from-pipe-ignored"
	smallEchoPayload     = "hi"
	binaryEnvelopePayload = "before\x00after"
	largePayloadSize     = 5000
)

func sessionCacheDir(sessionID string) string {
	return filepath.Join(os.TempDir(), "remote-agent-paste-bin-doctest-"+sessionID)
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

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	if len(req.Args) == 0 {
		req.Args = []string{"paste-bin"}
	}
	if req.Token == "" {
		req.Token = lib.TestPassword
	}
	if d.DOCTEST_SESSION_ID == "" {
		t.Fatal("session id empty on session.Doctest")
	}
	return nil
}

func fileTransferDir(configHome string) string {
	return filepath.Join(configHome, "file-transfer")
}

func scratchJSONPath(configHome string) string {
	return filepath.Join(fileTransferDir(configHome), "scratch.json")
}

type scratchJSON struct {
	Content   string `json:"content"`
	UpdatedAt string `json:"updated_at"`
}

func prepareScratch(req *Request, configHome string) error {
	if !req.ScratchReset && req.ScratchSeed == nil {
		return nil
	}
	if req.ScratchReset {
		if err := os.Remove(scratchJSONPath(configHome)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("reset scratch.json: %w", err)
		}
	}
	if req.ScratchSeed != nil {
		if err := os.MkdirAll(fileTransferDir(configHome), 0o755); err != nil {
			return fmt.Errorf("mkdir file-transfer: %w", err)
		}
		updatedAt := req.ScratchSeed.UpdatedAt
		if updatedAt == "" {
			updatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		payload := scratchJSON{
			Content:   req.ScratchSeed.Content,
			UpdatedAt: updatedAt,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal scratch.json: %w", err)
		}
		if err := os.WriteFile(scratchJSONPath(configHome), data, 0o644); err != nil {
			return fmt.Errorf("write scratch.json: %w", err)
		}
	}
	return nil
}

func resetScratch(req *Request) {
	req.ScratchReset = true
	req.ScratchSeed = nil
}

func seedScratch(req *Request, content, updatedAt string) {
	req.ScratchReset = false
	req.ScratchSeed = &ScratchSeed{Content: content, UpdatedAt: updatedAt}
}

func deleteScratch(req *Request) {
	req.ScratchReset = true
	req.ScratchSeed = nil
}

func setReadTTY(t *testing.T, req *Request, extraFlags ...string) {
	t.Helper()
	req.PipedStdin = nil
	req.StdinBytes = nil
	req.Args = append([]string{"paste-bin"}, extraFlags...)
}

func setWritePipe(t *testing.T, req *Request, payload []byte, extraFlags ...string) {
	t.Helper()
	piped := true
	req.PipedStdin = &piped
	req.StdinBytes = payload
	req.Args = append([]string{"paste-bin"}, extraFlags...)
}

func setReadForcePiped(t *testing.T, req *Request, pipePayload []byte) {
	t.Helper()
	piped := true
	req.PipedStdin = &piped
	req.StdinBytes = pipePayload
	req.Args = []string{"paste-bin", "--read"}
}

func repeatByte(b byte, n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}

func decodeScratchContent(apiContent string) ([]byte, error) {
	if strings.HasPrefix(apiContent, scratchB64Prefix) {
		return base64.StdEncoding.DecodeString(apiContent[len(scratchB64Prefix):])
	}
	return []byte(apiContent), nil
}

func assertStdoutExactBytes(t *testing.T, stdout string, want []byte) {
	t.Helper()
	if stdout != string(want) {
		t.Fatalf("stdout bytes mismatch:\nwant len=%d %q\ngot len=%d %q", len(want), want, len(stdout), stdout)
	}
}

func assertStdoutEmpty(t *testing.T, stdout string) {
	t.Helper()
	if stdout != "" {
		t.Fatalf("expected silent stdout; got %q", stdout)
	}
}

func assertScratchContentExact(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("scratch content mismatch:\nwant: %q\ngot:  %q", want, got)
	}
}

func assertScratchContentEmpty(t *testing.T, got string) {
	t.Helper()
	if got != "" {
		t.Fatalf("expected empty scratch content; got %q", got)
	}
}

func assertScratchB64Envelope(t *testing.T, apiContent string, wantRaw []byte) {
	t.Helper()
	if !strings.HasPrefix(apiContent, scratchB64Prefix) {
		t.Fatalf("expected %q prefix; got %q", scratchB64Prefix, apiContent)
	}
	decoded, err := decodeScratchContent(apiContent)
	if err != nil {
		t.Fatalf("decode scratch envelope: %v", err)
	}
	if string(decoded) != string(wantRaw) {
		t.Fatalf("decoded scratch mismatch:\nwant: %q\ngot:  %q", wantRaw, decoded)
	}
}

func assertStdoutEndsWithNewlineWhenNonEmpty(t *testing.T, stdout string) {
	t.Helper()
	if stdout == "" {
		return
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
```