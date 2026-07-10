# Scenario

**Feature**: create worktree with task slug from description

```
# task "Fix Login" -> slug fix-login
CreateWorktree(..., task="Fix Login") -> path/branch include fix-login
```

## Preconditions

Clean main git repo.

## Steps

1. Create clean main as `ProjectPath`.
2. Set `Task` to `Fix Login`.
3. POST create.

## Context

REQUIREMENT scenario 6. Slugify: lowercase, non-alnum → `-` → `fix-login`.

```go
import "testing"

func Setup(t *testing.T, req *Request) error {
	req.ProjectPath = mkCleanMainRepo(t)
	req.Task = "Fix Login"
	req.OmitTask = false
	return nil
}
```
