# Scenario

**Feature**: manual tty-watch script with early `/status\r` does not surface usage

```
5 boot snapshots + /status\r + 5 post-status snapshots -> no status fields
```

## Preconditions

Documents the debugging anti-pattern (not production `FetchStatus`):

```sh
for ((i=0;i<5;i++)); do tty-watch snapshot debug-codex; sleep 1; done
tty-watch send debug-codex $'/status\r'
```

Uses production codex argv; timing/send format match the manual script.

## Steps

1. `Op=ttywatch-real`.
2. `TTYWatchMode=user-script`.
3. `BootPollCount=5`, `StatusPollCount=5`.

## Context

Negative contract: early `/status\r` while model may still load does not produce
usage fields. Production path is `wait-idle-production-status` + `FetchStatus`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.Op = "ttywatch-real"
	req.TTYWatchMode = "user-script"
	req.BootPollCount = 5
	req.StatusPollCount = 5
	return nil
}
```