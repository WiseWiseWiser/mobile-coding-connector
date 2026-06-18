# Scenario

**Feature**: frontend route navigation smoke tests

```
# browser opens a /home route against quick-test + Vite
leaf script.js -> Playwright -> BASE_URL + route -> DOM checks
```

## Preconditions

1. Quick-test server and Vite dev server are started by the root `Run` function.
2. The React app is served through the quick-test proxy with v2 routing enabled.

## Steps

1. Each child leaf navigates to a specific route under `/home` or `/`.
2. The leaf `script.js` fixture performs browser navigation and DOM checks.
3. The leaf `Assert` function verifies `Response.ScriptResult` and exit code.

## Context

This grouping covers frontend navigation smoke tests: verifying that core home
routes load without error and that the root path redirects to `/home`.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if req.ScriptPath == "" {
		req.ScriptPath = "script.js"
	}
	if req.TimeoutSecs <= 0 {
		req.TimeoutSecs = 90
	}
	return nil
}
```