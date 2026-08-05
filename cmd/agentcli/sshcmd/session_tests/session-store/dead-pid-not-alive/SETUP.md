# Scenario

**Feature**: Load treats session not Alive when ServePID process is dead

```
# PID liveness on Load
tests -> Save(Alive=true, ServePID=dead) -> Load
  -> session returned with Alive=false (or equivalent not-alive)
```

## Preconditions

- Scenario: `session-dead-pid`.
- SessionToSave: Alive=true with ServePID of a process that has exited.
- Root Setup helper `deadPID` kills a short-lived child and returns its PID.

## Steps

1. Obtain a dead PID via `deadPID(t)`.
2. Save session with Alive=true and that ServePID.
3. Load; Assert Loaded non-nil and !Alive.

## Context

- ServePID==0 skips process check (covered by save-load). Non-zero + missing
  process → treat not alive so P1 gate fails.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, d *session.Doctest, req *Request) error {
	t.Helper()
	_ = d
	req.Scenario = ScenarioSessionDeadPID
	pid := deadPID(t)
	req.SessionToSave = sampleSession(req.ProfileID, req.ConfigDir, 19001, pid, true)
	return nil
}
```
