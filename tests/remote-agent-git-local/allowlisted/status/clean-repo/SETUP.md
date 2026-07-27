# Scenario

**Feature**: clean repo status

```
git init main + Initial commit -> status -> On branch main, clean
```

## Preconditions

Clean worktree after one commit.

## Steps

1. Init on `main`, commit `README.md`.

## Context

REQUIREMENT leaf #1.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	setGitLocalArgs(t, req, dir, "status")
	req.UseCLI = true
	return nil
}
```