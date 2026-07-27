# Scenario

**Feature**: one clean main repo with no linked worktrees

```
# register clean main only
ListProjects -> one ProjectStatus, clean=true, worktrees empty
```

## Preconditions

Temp git repo on `main` with one commit and a clean worktree.

## Steps

1. Create clean main repo.
2. Register its absolute path in `projects.json` under `WrkHome`.
3. Invoke list.

## Context

REQUIREMENT scenario 2.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	main := mkCleanMainRepo(t)
	writeProjectsJSON(t, req.WrkHome, []string{main})
	return nil
}
```
