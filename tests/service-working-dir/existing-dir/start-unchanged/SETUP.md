# Scenario

**Feature**: pre-existing workingDir — start succeeds normally

```
# directory exists before server boot
MkdirAll(<tmp>/existing-wd) -> start -> pid > 0, status running
```

## Preconditions

Parent `existing-dir` setup: `workingDir` is created before `services.json` is written.

## Steps

1. Pre-create `<temp>/existing-wd`.
2. Start service and assert unchanged success path.

## Context

REQUIREMENT leaf: `existing-dir/start-unchanged`.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	workingDir := filepath.Join(req.TempBase, "existing-wd")
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return err
	}
	req.WorkingDir = workingDir
	req.Services = []ServiceSeed{
		workingDirService("svc-wd-exist-001", "sleep-existing-wd", workingDir),
	}
	req.TargetID = "svc-wd-exist-001"
	return nil
}
```