# Scenario

**Feature**: non-git directory rejected by server

```
remote-agent git -C <plain-dir> status -> dir is not a git repository
```

## Preconditions

Directory exists but has no `.git`.

## Steps

1. Create empty temp dir (no `git init`).
2. Run `git -C <dir> status`.

## Context

REQUIREMENT leaf #7.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkWorkDir(t)
	setGitLocalArgs(t, req, dir, "status")
	return nil
}
```