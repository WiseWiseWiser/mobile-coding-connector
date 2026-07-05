# Scenario

**Bug**: in-process codex fetch times out during slow silent TUI boot

```
daemon-like PATH + 30s silent fake TUI + short timeout -> agent/usage.Fetch -> status ready
```

## Preconditions

1. Default in-process fetcher (no injectable mock).
2. `PATH` stripped to daemon launchd PATH.
3. Fake TUI silent for 30s before prompt and status fields (real Codex cloud-config stall).

## Steps

1. `Op=fetch-inprocess`.
2. `StripDaemonPATH=true`.
3. `ShowStatusCommand` = 30s slow-boot fake TUI.
4. `FetchTimeoutSecs=5`.

## Context

Deterministic repro of `timeout waiting for snapshot frame` during slow boot.
`slow-boot-snapshot` uses 12s silence with 60s timeout (passes); this leaf uses
a shorter timeout to force failure.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "fetch-inprocess"
	req.StripDaemonPATH = true
	req.ShowStatusCommand = slowBootFakeCodexTUIWithDelay(30)
	req.FetchTimeoutSecs = 5
	return nil
}
```