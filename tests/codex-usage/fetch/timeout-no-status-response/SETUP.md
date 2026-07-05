# Scenario

**Feature**: in-process fetch errors when TUI never prints /status fields

```
never-respond fake TUI -> agent/usage.Fetch -> status error (timeout)
```

## Preconditions

1. Default in-process fetcher (no injectable mock).
2. `PATH` stripped to daemon launchd PATH.
3. Fake TUI prints Codex prompt immediately but sleeps after `/status` (no parseable output).

## Steps

1. `Op=fetch-inprocess`.
2. `StripDaemonPATH=true`.
3. `ShowStatusCommand` = never-respond fake TUI.
4. `FetchTimeoutSecs=5` (service ctx still caps at 90s).

## Context

Negative contract: menu-bar error `timeout waiting for status output` when Codex
never renders usage. Happy path covered by `slow-boot-snapshot` and
`real-codex-inprocess`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "fetch-inprocess"
	req.StripDaemonPATH = true
	req.ShowStatusCommand = neverRespondFakeCodexTUI()
	req.FetchTimeoutSecs = 5
	return nil
}
```