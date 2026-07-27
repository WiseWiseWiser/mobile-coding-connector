# Scenario

**Feature**: keep-alive daemon sees core HTTP readiness before slow extension startup

```
# daemon spawns server, polls /ping within 10s; server core binds then async extension
keep-alive --port P -> managed server -> [bootstrap] core_ready -> /ping
keep-alive <- port ready (no restart loop) <- extension tasks (delayed via test hook)
```

## Preconditions

1. `go build` can produce the `ai-critic` binary from the repo root.
2. Isolated `AI_CRITIC_HOME` with test credentials (created in `Run`).
3. Keep-alive management HTTP port (`23312`) is singleton — tests acquire a
   file lock for the doctest session to avoid parallel daemon collisions.
4. Extension test hooks are driven by env vars (no real Cloudflare API calls).

## Steps

1. Root `Run` builds the binary, creates config home, starts `keep-alive` with
   explicit `--port` and `--credentials-file` server args.
2. Polls `/ping` during `ObserveSecs`, then tears down the daemon process group.
3. Merges daemon log file, captured stdout, and `ai-critic-server.log` for parsing.

## Context

Targets the bug where `RunSideEffectTasks()` runs before `server.Serve()`, blocking
port bind until tunnels finish. Complements `tests/server/` (opencode auto-start
semantics) without modifying that tree.

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	unlock := acquireKeepAliveLock(t, d)
	t.Cleanup(unlock)

	if req.ServerPort <= 0 {
		req.ServerPort = 23712
	}
	if req.ObserveSecs <= 0 {
		req.ObserveSecs = 12
	}
	return nil
}

func acquireKeepAliveLock(t *testing.T, d *session.Doctest) func() {
	// Port 23312 is a process-wide singleton. Wait for exclusive flock so
	// parallel leaves serialize instead of t.Skip.
	sid := ""
	if d != nil {
		sid = d.DOCTEST_SESSION_ID
	}
	if sid == "" {
		sid = "default"
	}
	_ = sid
	lockPath := filepath.Join(os.TempDir(), "ai-critic-keepalive-doctest-global.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("open keep-alive lock %s: %v", lockPath, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		t.Fatalf("flock keep-alive lock %s: %v", lockPath, err)
	}
	_, _ = f.WriteAt([]byte(fmt.Sprintf("%d\n", os.Getpid())), 0)
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}
}
```