# Scenario

**Feature**: create worktree success opens returned path then refreshes

```
createWorktree success -> openITerm2(created.path, reuse) -> refreshProjects()
```

## Preconditions

`createWorktree` / `createWrkWorktree` flow in AppState or menu.

## Steps

1. Set `ClientLeaf=create-worktree-opens`.

## Context

REQUIREMENT: after create worktree → open new path with reuse, then refresh.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.ClientLeaf = "create-worktree-opens"
	return nil
}
```
