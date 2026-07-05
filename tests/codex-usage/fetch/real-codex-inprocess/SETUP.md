# Scenario

**Feature**: real codex CLI in-process fetch under daemon-like PATH

```
login-shell codex resolve + daemon PATH + node bin dir -> agent/usage.Fetch -> status ready
```

## Preconditions

1. Real `codex` installed (resolved via `bash -lic 'command -v codex'`).
2. No `CODEX_SHOW_STATUS_COMMAND` hook — production `buildCodexArgv` path.
3. `PATH` stripped to launchd PATH plus codex's `bin` directory (node shim support).
4. Isolated `TTY_WATCH_HOME` per attempt.

## Steps

1. `Op=fetch-inprocess`.
2. `UseRealCodex=true`.
3. `StripDaemonPATH=true`.
4. `RealCodexAttempts=5` (surfaces intermittent snapshot/status timeouts).
5. `FetchTimeoutSecs=90`.

## Context

Reproduces menu-bar `timeout waiting for snapshot frame` / `timeout waiting for status output`
against the real Codex TUI boot (cloud-config stall, model loading). Synthetic
`slow-boot-snapshot` remains a deterministic unit; this leaf uses the real CLI.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "fetch-inprocess"
	req.UseRealCodex = true
	req.StripDaemonPATH = true
	req.RealCodexAttempts = 5
	req.FetchTimeoutSecs = 90
	return nil
}
```