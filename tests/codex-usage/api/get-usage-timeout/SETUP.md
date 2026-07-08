# Scenario

**Feature**: GET /api/codex/usage returns error when TUI never prints /status fields

```
never-respond fake TUI -> server fetch -> GET :23712/api/codex/usage -> status error
```

## Preconditions

Fake Codex TUI via `CODEX_SHOW_STATUS_COMMAND` that never renders parseable status.

## Steps

1. `Op=api`.
2. `ShowStatusCommand` = never-respond fake TUI.
3. `FetchTimeoutSecs=5`.
4. `WaitAPIReadySecs=95` (allow error cache to populate).

## Context

Negative API contract mirroring `fetch/timeout-no-status-response`. Requires
`ai-critic-react/dist` for daemon build.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.Op = "api"
	req.TTYWatchHome = filepath.Join(t.TempDir(), ".tty-watch")
	req.ShowStatusCommand = neverRespondFakeCodexTUI()
	req.FetchTimeoutSecs = 5
	req.WaitAPIReadySecs = 95
	req.WaitAPIError = true
	return nil
}
```