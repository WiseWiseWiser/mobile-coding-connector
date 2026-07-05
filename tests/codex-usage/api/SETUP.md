# Scenario

**Feature**: HTTP API for codex usage on keep-alive daemon

```
keep-alive + CODEX_SHOW_STATUS_BIN -> GET /api/codex/usage
```

## Preconditions

Daemon exposes codex usage route; session lock held.

## Steps

1. Set `Op=api` in leaf.

## Context

End-to-end API contract for Swift menu-bar client.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.Op = "api"
	req.TTYWatchHome = filepath.Join(t.TempDir(), ".tty-watch")
	return nil
}
```