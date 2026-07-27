# Scenario

**Feature**: clean git repo shows branch, commit, and clean worktree

```
# git init on main, Initial commit, register project
remote-agent project list -> Git Branch: main, Git Commit: <hash>  Initial commit, Worktree: clean
```

## Preconditions

Empty temp directory; `git` available.

## Steps

1. Create temp project dir with `git init` and `git branch -M main`.
2. Add `README.md` and commit with message `Initial commit`.
3. Register project `clean-repo-test` (`clean-001`) pointing at the repo dir.

## Context

REQUIREMENT leaf: `clean-repo/` — clean repo with "Initial commit".

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	dir := mkProjectDir(t)
	gitInitWithMain(t, dir)
	gitInitialCommit(t, dir, "Initial commit")

	req.Project = ProjectEntry{
		ID:   "clean-001",
		Name: "clean-repo-test",
		Dir:  dir,
	}
	req.UseCLI = true
	return nil
}
```