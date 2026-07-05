# Scenario

**Feature**: real tty-watch CLI timing against live Codex TUI

```
tty-watch run/list/snapshot/send/kill + real codex -> prompt/status timing transcript
```

## Preconditions

1. `tty-watch` and `codex` on PATH (login-shell or `CODEX_USAGE_TEST_CODEX_PATH`).
2. Isolated `TTY_WATCH_HOME` per run.
3. Production Codex argv: `--dangerously-bypass-approvals-and-sandbox -c mcp_servers={}`.

## Steps

1. Set `Op=ttywatch-real` in leaves.
2. Leaf sets `TTYWatchMode` (`user-script` or `wait-idle-production`).

## Context

Phase-A real-world performance capture from manual tty-watch experiments (2026-07-05).
Documents why early `/status\r` after 5 boot snapshots fails while wait-idle +
`/status\n\r` succeeds in ~16s.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	if req.Op == "" {
		req.Op = "ttywatch-real"
	}
	return nil
}
```