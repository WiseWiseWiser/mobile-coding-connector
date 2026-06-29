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
	"testing"
	"time"
)

func Setup(t *testing.T, req *Request) error {
	unlock := acquireKeepAliveLock(t)
	t.Cleanup(unlock)

	if req.ServerPort <= 0 {
		req.ServerPort = 23712
	}
	if req.ObserveSecs <= 0 {
		req.ObserveSecs = 12
	}
	return nil
}

func acquireKeepAliveLock(t *testing.T) func() {
	session := os.Getenv("DOCTEST_SESSION_ID")
	if session == "" {
		session = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	lockPath := filepath.Join(os.TempDir(), "ai-critic-keepalive-"+session+".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Skipf("another keep-alive doctest holds lock %s: %v", lockPath, err)
		return func() {}
	}
	_, _ = f.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
	_ = f.Close()
	return func() { os.Remove(lockPath) }
}
```