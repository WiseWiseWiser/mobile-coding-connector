# Scenario

**Feature**: grok usage parse, fetch, API, and refresh overlap

```
mock GROK_SHOW_USAGE_BIN -> parser/service -> GET /api/grok/usage (optional daemon)
```

## Preconditions

1. `macosapp/grokusage` package provides parser and `TestExported_*` service hooks.
2. Keep-alive daemon registers `GET /api/grok/usage` for API leaves.
3. Mock scripts live in `tests/grok-usage/testdata/` and are chmod +x before exec.
4. API leaves acquire keep-alive session lock (port `23312`).

## Steps

1. Root `Setup` sets defaults and lock for API/refresh paths.
2. Leaf `Setup` sets `Op`, fixtures, and mock script names.
3. Root `Run` dispatches by `Op` to parse, fetch, HTTP, or overlap harness.
4. Leaf `Assert` checks parsed fields, service status, or API JSON.

## Context

Implements REQUIREMENT-DESIGN-macos-app-and-bar.md Feature 2. Live
`debug-grok-show-usage` is out of scope (tag `slow` if added later).

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