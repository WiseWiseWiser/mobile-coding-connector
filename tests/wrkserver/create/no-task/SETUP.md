# Scenario

**Feature**: create worktree without task slug

```
# omit task field
CreateWorktree(project_path) -> path under $WRK_HOME/worktrees, branch without task slug
```

## Preconditions

Clean main git repo; `OmitTask=true`.

## Steps

1. Create clean main repo as `ProjectPath`.
2. Omit `task` from JSON body.
3. POST create.

## Context

REQUIREMENT scenario 5.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ProjectPath = mkCleanMainRepo(t)
	req.OmitTask = true
	return nil
}
```
