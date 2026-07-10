# Scenario

**Feature**: CLI schedule flags — `--every` | `--cron` (local convert) | `--cron-utc`

```
# remote-agent cron add --cron "…" under fixed TZ
CLI local wall time -> convert to UTC when safe -> POST create
# unsafe convert -> error mentioning --cron-utc
```

## Preconditions

1. `remote-agent` binary from session cache.
2. Leaves set `CLIEnv` for deterministic TZ (fixed offset or complex expr).
3. Mutually exclusive schedule flags on add/update.

## Steps

1. Leaf sets `UseCLI`, `CLIArgs`, optional `CLIEnv`.
2. Run executes CLI against live server.
3. Assert checks exit code, stdout convert messages, or stored UTC expr via list.

## Context

Priority leaves 8–9 (CLI `--cron` convert + unsafe error).

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.UseCLI = true
	req.Action = "create"
	return nil
}
```
