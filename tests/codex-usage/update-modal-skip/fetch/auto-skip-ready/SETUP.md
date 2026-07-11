# Scenario

**Bug**: menu bar Codex usage times out on Update available; must auto-Skip and fetch

```
fake-tui-auto-skip.py -> CSI Down + Enter (production) -> status ready with usage fields
```

## Preconditions

1. `fake-tui-auto-skip.py` starts on Update now, moves to Skip on CSI Down, idle on Enter.
2. Daemon-like PATH + isolated `TTY_WATCH_HOME`.

## Steps

1. `ShowStatusCommand` = auto-skip fake.
2. `SessionID` unique per leaf.
3. `FetchTimeoutSecs=30` (service ctx still caps at 90s).

## Context

Happy path for REQUIREMENT auto-Skip. Fast once implemented; before fix fails with
timeout waiting for status output / prompt.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.ShowStatusCommand = autoSkipFakeCommand()
	req.SessionID = "codex-update-modal-auto-skip"
	req.MarkerDir = filepath.Join(t.TempDir(), "markers")
	req.FetchTimeoutSecs = 30
	return nil
}
```
