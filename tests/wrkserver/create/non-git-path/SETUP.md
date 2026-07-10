# Scenario

**Feature**: create rejects non-git project_path

```
# plain directory (not a git repo)
CreateWorktree(project_path=plainDir) -> 4xx {"error":"..."}
```

## Preconditions

Temp directory that is not a git repository.

## Steps

1. Create plain temp dir as `ProjectPath`.
2. Omit task.
3. POST create.

## Context

REQUIREMENT scenario 9.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ProjectPath = mkTempDir(t, "wrkserver-nongit-*")
	req.OmitTask = true
	return nil
}
```
