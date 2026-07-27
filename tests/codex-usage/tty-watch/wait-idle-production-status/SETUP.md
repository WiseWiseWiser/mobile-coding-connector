# Scenario

**Feature**: production-like tty-watch flow completes within ~20s

```
wait model idle -> send /status\n\r -> status fields on screen
```

## Preconditions

Real codex + production argv; poll until prompt idle before `/status`.

## Steps

1. `Op=ttywatch-real`.
2. `TTYWatchMode=wait-idle-production`.
3. `MaxWaitSecs=30`.

## Context

Observed 2026-07-05: prompt idle ~5s, status fields at +16s total.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "ttywatch-real"
	req.TTYWatchMode = "wait-idle-production"
	req.MaxWaitSecs = 30
	return nil
}
```