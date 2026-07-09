# Scenario

**Feature**: missing flat workingDir is created and service runs

```
# single-segment missing path
services.json(workingDir=<tmp>/my-svc-wd) -> start -> Stat is dir, pid > 0
```

## Preconditions

Parent `missing-dir` setup: `workingDir` not pre-created.

## Steps

1. Use flat path `<temp>/my-svc-wd` (not created).
2. Start service and assert directory exists and process is running.

## Context

REQUIREMENT leaf: `missing-dir/creates-and-runs`.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	workingDir := filepath.Join(req.TempBase, "my-svc-wd")
	req.WorkingDir = workingDir
	req.Services = []ServiceSeed{
		workingDirService("svc-wd-create-001", "sleep-missing-wd", workingDir),
	}
	req.TargetID = "svc-wd-create-001"
	return nil
}
```