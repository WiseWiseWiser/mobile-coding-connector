# Scenario

**Feature**: grok usage parse, in-process fetch, API, and refresh overlap

```
injectable usage.Fetch hook -> parser/service -> GET /api/grok/usage (optional daemon)
```

## Preconditions

1. `macosapp/grokusage` provides parser and `TestExported_NewService`,
   `TestExported_SetFetcher`, `TestExported_FetchOnce`, `TestExported_TriggerRefresh`.
2. Service calls `agent/usage.Fetch(ctx, Grok)` by default (no `GROK_SHOW_USAGE_BIN`).
3. Keep-alive daemon registers `GET /api/grok/usage` for API leaves.
4. API leaves use `GROK_SHOW_USAGE_COMMAND` fake TUI hook.
5. API leaves acquire keep-alive session lock (port `23312`).

## Steps

1. Root `Setup` sets defaults and lock for API/refresh paths.
2. Leaf `Setup` sets `Op`, fixtures, or `FetchMode`.
3. Root `Run` dispatches by `Op` to parse, fetch, HTTP, or overlap harness.
4. Leaf `Assert` checks parsed fields, service status, or API JSON.

## Context

Implements REQUIREMENT-DESIGN-in-process-usage-fetch.md Part B. Live Grok TTY is
out of scope (tag `slow` if added later).

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
	session := os.Getenv("DOCTEST_SESSION_ID")
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