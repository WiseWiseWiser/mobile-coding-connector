# Scenario

**Feature**: start with an existing `workingDir` behaves unchanged

```
# workingDir already on disk before start
MkdirAll(workingDir) -> services.json -> POST /api/services/start -> running
```

## Preconditions

1. `workingDir` is pre-created in leaf setup before the server starts.
2. Service command is `sleep 300`.

## Steps

1. Leaf setup creates `workingDir` with `os.MkdirAll`.
2. Seed `services.json` and start the service.
3. Assert normal running checks (no regression).

## Context

Sibling `missing-dir/` covers auto-creation when the path is absent.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.TempBase = t.TempDir()
	return nil
}
```