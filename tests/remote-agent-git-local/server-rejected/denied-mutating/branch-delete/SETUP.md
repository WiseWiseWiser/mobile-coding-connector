# Scenario

**Feature**: destructive `git branch -D` denied

```
remote-agent git -C <repo> branch -D extra -> denied
```

## Preconditions

Repo with branches `main` and `extra`.

## Steps

1. Create `extra` branch locally via harness git.
2. Attempt `branch -D extra` via remote-agent.

## Context

Requirement: branch list/show only; deny delete.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkDeniedRepo(t)
	gitRun(t, dir, "branch", "extra")
	setGitLocalArgs(t, req, dir, "branch", "-D", "extra")
	return nil
}
```