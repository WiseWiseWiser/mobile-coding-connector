# Scenario

**Feature**: GET /api/codex/usage returns ready JSON

```
server in-process fetch (CODEX_SHOW_STATUS_COMMAND) -> GET :23712/api/codex/usage -> status ready
```

## Preconditions

Fake Codex TUI via `CODEX_SHOW_STATUS_COMMAND` and isolated `TTY_WATCH_HOME`.

## Steps

1. `WaitAPIReadySecs=15`.
2. Default fake TUI from root `Run` when `ShowStatusCommand` empty.

## Context

REQUIREMENT leaf: `api/get-usage-ready`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.WaitAPIReadySecs = 15
	return nil
}
```