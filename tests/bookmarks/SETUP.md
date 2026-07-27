# Scenario

**Feature**: local-agent bookmarks tree — store, HTTP API, CLI, resolve, menu labels

```
# isolated bookmarks.json + Manager
leaf Setup -> Mode/op params
  -> store | ResolveBrowser | RegisterAPIWithManager | agentcli bookmarks | labels
  -> DocView / HTTP / stdout asserts
```

## Preconditions

1. Product package `server/bookmarks` and CLI `bookmarks` are the system under test
   (missing → compile or Run failures = RED in Classic TDD).
2. Each leaf isolates store path (temp dir); never writes the developer HOME.
3. API leaves wrap mux with Bearer `req.Token`.
4. CLI leaves use `agentcli.Run(LocalProfile())` against httptest API when needed.

## Steps

1. Root `Setup` sets default token.
2. Leaf `Setup` sets `Mode` and operation fields.
3. Root `Run` dispatches on `Mode` and fills `Response`.
4. Leaf `Assert` checks document shape, HTTP status, CLI output, or pure results.

## Context

REQUIREMENT-DESIGN-bookmarks-management.md. Locked: file storage + API, Chrome
tree, optional per-bookmark browser, CLI full CRUD + open, menu read-only v1,
no Chrome import, no remote menu bar.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	if req.Token == "" {
		req.Token = "test-token"
	}
	return nil
}
```
