# Scenario

**Feature**: `git log --oneline` shows newest first

```
two commits -> log --oneline -2 -> two lines, Second before Initial
```

## Preconditions

Two commits on `main`.

## Context

REQUIREMENT leaf #4.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	gitSecondCommit(t, dir, "second.txt", "Second commit")
	setGitLocalArgs(t, req, dir, "log", "--oneline", "-2")
	return nil
}
```