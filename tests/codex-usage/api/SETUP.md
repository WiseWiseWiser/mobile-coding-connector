# Scenario

**Feature**: HTTP API for codex usage on main server port

```
keep-alive spawns server + CODEX_SHOW_STATUS_COMMAND -> GET :23712/api/codex/usage
```

## Preconditions

Server exposes codex usage route on port `23712`; session lock held.

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