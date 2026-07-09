# Scenario

**Feature**: start creates a missing `workingDir` before launching the service

```
# workingDir path configured but absent on disk
services.json(workingDir=/tmp/.../missing) -> POST /api/services/start

# server must mkdir workingDir then exec sleep
ensureServiceWorkingDir -> bash -lc sleep 300 -> pid > 0
```

## Preconditions

1. `workingDir` is set in `services.json` but **not** created in setup.
2. Parent temp base exists only as the anchor for the eventual path.
3. Service command is `sleep 300` for stable PID checks.

## Steps

1. Leaf setup picks a non-existent `workingDir` under `t.TempDir()`.
2. Seed `services.json` with that `workingDir`.
3. `POST /api/services/start` and verify directory + running state.

## Context

Sibling `existing-dir/` covers the pre-existing directory case.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.TempBase = t.TempDir()
	return nil
}
```