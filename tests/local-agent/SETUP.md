# Scenario

**Feature**: local-agent CLI harness (L2 mass + sparse L3 smokes)

```
# L2: agentcli.Run + optional in-process mux / testhooks overrides | L3 UseCLI: product binaries
leaf Setup -> optional server -> local-agent -> stdout/stderr + config side effects
```

## Preconditions

1. Doctest injects `DOCTEST_SESSION_ID` to scope a file cache under
   `$TMPDIR/local-agent-doctest-<session>/` (L3 binaries when needed).
2. Session file locks (`flock`) serialize first-time cache population across parallel leaves.
3. **L2** (default): `agentcli.Run(LocalProfile())` with testhooks home/port/reachability
   overrides and optional in-process `/ping` + auth stubs (no product binary exec; no Setenv).
4. **L3** (`UseCLI` smokes only): product `local-agent` (+ optional `ai-critic-server`) with
   child env HOME and testhooks env vars.
5. Top-level `-h` stays L3 (less-gen `os.Exit` on help).

## Steps

1. Leaf `Setup` configures flags, seeds, and optional `UseCLI`.
2. Root `Run` prepares agent home/config; L2 starts optional in-process server stubs.
3. Executes via agentcli (L2) or product binary (L3).
4. Captures stdout, stderr, exit code; optional config snapshots.

## Context

Implements REQUIREMENT-DESIGN-local-agent-cli.md. L2 covers resolution, auth guidance,
and config isolation; sparse L3 smokes keep binary + help paths.

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/doctest/session"
)


func sessionCacheDir(sessionID string) string {
	return filepath.Join(os.TempDir(), "local-agent-doctest-"+sessionID)
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
	agentBin = filepath.Join(cacheDir, "local-agent")
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
			{agentBin, "./cmd/local-agent"},
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
	if d.DOCTEST_SESSION_ID == "" {
		t.Fatal("session id empty on session.Doctest")
	}
	if req.Args == nil {
		req.Args = []string{}
	}
	return nil
}
```