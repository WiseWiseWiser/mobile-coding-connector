# Scenario

**Feature**: unknown git subcommand rejected at CLI

```
remote-agent git -C <repo> frobnicate -> unknown subcommand, no HTTP
```

## Preconditions

Valid git repository on disk.

## Steps

1. Init repo with initial commit.
2. Invoke `git -C <dir> frobnicate`.

## Context

REQUIREMENT leaf #8.

```go
import (
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkWorkDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")
	setGitLocalArgs(t, req, dir, "frobnicate")
	return nil
}
```