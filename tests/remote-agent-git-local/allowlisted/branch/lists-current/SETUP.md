# Scenario

**Feature**: current branch marked in branch list

```
repo on main -> branch -> * main
```

## Context

REQUIREMENT leaf #5.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	setGitLocalArgs(t, req, dir, "branch")
	return nil
}
```