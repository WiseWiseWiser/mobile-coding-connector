# Scenario

**Feature**: grok usage parse, library fetch, API, and refresh overlap

```
GROK_SHOW_USAGE_COMMAND fake TUI -> tty.FetchUsageWithOptions -> service cache -> GET /api/grok/usage (optional daemon)
```

## Preconditions

1. `agent/grok/tty` provides `ParseShowUsageOutput` and `FetchUsageWithOptions`.
2. `macosapp/grokusage` delegates fetch to `tty` and exposes `TestExported_*` hooks.
3. Mock fake-TUI scripts live in `tests/grok-usage/testdata/` (chmod +x before use).
4. `GET /api/grok/usage` is served on main server port `23712` (not daemon `23312`).
5. API leaves start keep-alive (spawns server) and acquire session lock on `23312`.

## Steps

1. Root `Setup` sets defaults and lock for API/refresh paths.
2. Leaf `Setup` sets `Op`, fixtures, and mock script names.
3. Root `Run` dispatches by `Op` to parse, fetch, HTTP, or overlap harness.
4. Leaf `Assert` checks parsed fields, service status, or API JSON.

## Context

Implements REQUIREMENT-DESIGN-grok-tty-show-usage.md. Live grok PTY fetch is out
of scope (tag `slow` if added later).

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func Setup(t *testing.T, req *Request) error {
	if req.WaitAPIReadySecs <= 0 {
		req.WaitAPIReadySecs = 12
	}
	if req.Op == "api" {
		unlock := acquireKeepAliveLock(t)
		t.Cleanup(unlock)
	}
	return nil
}

func acquireKeepAliveLock(t *testing.T) func() {
	session := DOCTEST_SESSION_ID
	if session == "" {
		session = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	lockPath := filepath.Join(os.TempDir(), "ai-critic-grok-usage-"+session+".lock")
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