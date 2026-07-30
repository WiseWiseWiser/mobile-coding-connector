# Scenario

**Feature**: service add --start starts the process

```
service add --name demo-start --command "sleep 30" --start
  -> exit 0; status running / PID > 0
```

## Preconditions

1. Long-lived command so PID remains after short settle wait.
2. Product `add --start` calls Start after Save.

## Steps

1. Run add with `--start` and `sleep 30`.
2. Wait ~1s for process table to settle.
3. Assert running / PID > 0.

## Context

REQUIREMENT leaf: `add/--start`. Prefer long-lived `sleep`, not `true`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.TargetName = "demo-start"
	req.WaitAfterSecs = 1
	setCLI(req,
		"service", "add",
		"--name", "demo-start",
		"--command", "sleep 30",
		"--start",
	)
	return nil
}
```
