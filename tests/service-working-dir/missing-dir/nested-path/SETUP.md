# Scenario

**Feature**: missing nested workingDir path is created recursively

```
# deep nested path absent on disk
services.json(workingDir=<tmp>/a/b/c) -> start -> all parents created
```

## Preconditions

Parent `missing-dir` setup: nested `a/b/c` segments are **not** pre-created.

## Steps

1. Configure `workingDir` as `<temp>/a/b/c`.
2. Start service and assert full nested path exists and service is running.

## Context

REQUIREMENT leaf: `missing-dir/nested-path`.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	workingDir := filepath.Join(req.TempBase, "a", "b", "c")
	req.WorkingDir = workingDir
	req.Services = []ServiceSeed{
		workingDirService("svc-wd-nested-001", "sleep-nested-wd", workingDir),
	}
	req.TargetID = "svc-wd-nested-001"
	return nil
}
```