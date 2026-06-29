# Scenario

**Feature**: detached HEAD shows `(detached)` branch label

```
# commit on main, git checkout --detach HEAD
remote-agent project list -> Git Branch: (detached), commit still shown, Worktree: clean
```

## Preconditions

Git repo with at least one commit.

## Steps

1. Create repo on `main` with commit message `Initial commit`.
2. Run `git checkout --detach HEAD`.
3. Register project `detached-head-test` (`detached-001`).

## Context

REQUIREMENT leaf: `detached-head/` — detached HEAD, branch `(detached)`.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkProjectDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	gitRun(t, dir, "checkout", "--detach", "HEAD")

	req.Project = ProjectEntry{
		ID:   "detached-001",
		Name: "detached-head-test",
		Dir:  dir,
	}
	return nil
}
```