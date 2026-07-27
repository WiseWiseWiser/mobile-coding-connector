# Scenario

**Feature**: POST create worktree handler

```
# create linked worktree under $WRK_HOME/worktrees
CreateWorktree(project_path, task?) -> 200 {path,branch} | 4xx {error}
```

## Preconditions

`Op=create` dispatches to `Server.CreateWorktree` via httptest.

## Steps

1. Set `Op` to `create`.
2. Leaf supplies `ProjectPath`, optional `Task` / `OmitTask`.

## Context

REQUIREMENT scenarios 5–9. Whitespace-only task is treated as empty (no slug),
even if CLI `wrk --task` rejects whitespace.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "create"
	return nil
}
```
