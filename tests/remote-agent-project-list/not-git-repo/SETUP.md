# Scenario

**Feature**: non-git directory shows dashes for git status lines

```
# plain directory without git init
remote-agent project list -> Git Branch/Commit/Worktree: -
```

## Preconditions

Temp directory exists; no `.git` metadata.

## Steps

1. Create temp project dir and write `plain.txt` (no `git init`).
2. Register project `not-git-test` (`not-git-001`).

## Context

REQUIREMENT leaf: `not-git-repo/` — plain dir, dashes for git lines.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkProjectDir(t)
	if err := os.WriteFile(filepath.Join(dir, "plain.txt"), []byte("not a git repo\n"), 0644); err != nil {
		return err
	}

	req.Project = ProjectEntry{
		ID:   "not-git-001",
		Name: "not-git-test",
		Dir:  dir,
	}
	return nil
}
```