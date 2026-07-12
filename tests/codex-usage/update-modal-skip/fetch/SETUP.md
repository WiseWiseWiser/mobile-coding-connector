# Scenario

**Feature**: in-process FetchStatus auto-Skip via CODEX_SHOW_STATUS_COMMAND fake TUI

```
fake update-menu TUI -> waitForPrompt Skip protocol -> /status -> service ready
```

## Preconditions

1. Default in-process fetcher (no injectable mock).
2. `PATH` stripped to daemon launchd PATH so only the explicit python fake runs.
3. Fake scripts under `testdata/update-modal-skip/`.

## Steps

1. Set `Op=fetch-inprocess`.
2. Leaf sets ShowStatusCommand (auto-skip or stuck) and MarkerDir as needed.

## Context

End-to-end contract for production Skip keys (CSI Down + Enter). RED until
`waitForPrompt` implements the protocol and writable banner gate is narrowed.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	req.Op = "fetch-inprocess"
	req.StripDaemonPATH = true
	if req.TTYWatchHome == "" {
		req.TTYWatchHome = filepath.Join(t.TempDir(), ".tty-watch")
	}
	if req.FetchTimeoutSecs <= 0 {
		req.FetchTimeoutSecs = 30
	}
	return nil
}
```
