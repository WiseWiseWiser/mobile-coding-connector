# Scenario

**Feature**: in-process codex fetch survives slow silent TUI boot

```
daemon-like PATH + slow CODEX_SHOW_STATUS_COMMAND -> agent/usage.Fetch -> status ready
```

## Preconditions

1. Default in-process fetcher (no injectable mock).
2. `PATH` stripped to daemon launchd PATH.
3. Fake TUI silent for 12s, then prints Codex prompt and `/status` fields.

## Steps

1. `Op=fetch-inprocess`.
2. `StripDaemonPATH=true`.
3. `FetchTimeoutSecs=60`.

## Context

Deterministic synthetic repro of snapshot-frame timeout mechanics (silent fake TUI).
For the real Codex CLI path, see `fetch/real-codex-inprocess` (tag `slow`).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "fetch-inprocess"
	req.StripDaemonPATH = true
	req.FetchTimeoutSecs = 60
	return nil
}
```