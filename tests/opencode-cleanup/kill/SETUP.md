# Scenario

**Feature**: KillOpencodeServePIDs and CleanupOpencodeServe

```
ps verify opencode serve -> SIGTERM -> registry clear
wrong process PID -> skipped, process survives
```

## Preconditions

- `KillOpencodeServePIDs` must not use `pkill -f`.
- Process command verified via `ps -p PID -o command=`.

## Steps

1. Set `Op = OpKill` or `OpCleanup` per leaf.

## Context

MECE: reject wrong process, kill real child, clear registry after cleanup.

```go
import (
	"os/exec"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof required for kill helper tests")
	}
	return nil
}
```
