# Scenario

**Feature**: grok usage parse, HTTP fetch, API, and refresh overlap

```
injectable/HTTP fetch -> service cache -> GET /api/grok/usage (optional daemon)
```

## Preconditions

1. `agent/grok/tty` provides `ParseShowUsageOutput` (parse leaves).
2. `macosapp/grokusage` default fetcher uses `dot-pkgs/shell/grok/usage` HTTP billing;
   tests inject via `TestExported_SetFetcher` / `AI_CRITIC_GROK_USAGE_FIXTURE`.
3. `GET /api/grok/usage` is served on main server port `23712` (not daemon `23312`).
4. API leaves start keep-alive (spawns server) and acquire session lock on `23312`.

## Steps

1. Root `Setup` sets defaults and lock for API/refresh paths.
2. Leaf `Setup` sets `Op`, fixtures, and fetch modes.
3. Root `Run` dispatches by `Op` to parse, fetch, HTTP, or overlap harness.
4. Leaf `Assert` checks parsed fields, service status, or API JSON.

## Context

Live grok PTY fetch is out of scope.

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	if req.WaitAPIReadySecs <= 0 {
		req.WaitAPIReadySecs = 12
	}
	if req.Op == "api" {
		unlock := acquireKeepAliveLock(t, d)
		t.Cleanup(unlock)
	}
	return nil
}

func acquireKeepAliveLock(t *testing.T, d *session.Doctest) func() {
	sid := ""
	if d != nil {
		sid = d.DOCTEST_SESSION_ID
	}
	if sid == "" {
		sid = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	lockPath := filepath.Join(os.TempDir(), "ai-critic-grok-usage-"+sid+".lock")
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
