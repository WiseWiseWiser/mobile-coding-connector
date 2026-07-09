# Scenario

**Bug**: missing workingDir must not produce misleading bash fork/exec error

```
# workingDir absent — without MkdirAll fix, cmd.Dir breaks exec
services.json(workingDir=<tmp>/openclaw-like) -> start

# log should show normal start, not fork/exec /bin/bash
services/{id}.log contains "starting service", not "fork/exec /bin/bash"
```

## Preconditions

Parent `missing-dir` setup: `workingDir` not pre-created.

## Steps

1. Use path `<temp>/my-openclaw` (mirrors openclaw bug report).
2. Start service and inspect service log tail.

## Context

REQUIREMENT leaf: `missing-dir/no-bash-fork-error`. Guards against the
misleading Go error when `cmd.Dir` points at a missing path.

```go
import (
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	workingDir := filepath.Join(req.TempBase, "my-openclaw")
	req.WorkingDir = workingDir
	req.Services = []ServiceSeed{
		workingDirService("svc-wd-fork-001", "sleep-fork-check", workingDir),
	}
	req.TargetID = "svc-wd-fork-001"
	return nil
}
```