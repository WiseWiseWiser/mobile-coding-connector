# Scenario

**Feature**: main plus linked worktrees with dirty flag

```
# main clean; one linked worktree dirty
ListProjects -> worktrees[] with clean flags; linked is_main=false
```

## Preconditions

Main repo with initial commit; one linked worktree via `git worktree add` with
an uncommitted file (dirty).

## Steps

1. Create clean main on `main`.
2. Add linked worktree on branch `feat-dirty` beside the main dir.
3. Write an untracked/modified file in the linked worktree (dirty).
4. Register main path in `projects.json`.

## Context

REQUIREMENT scenario 3. Prefer linked-list semantics (exclude main as a linked
row; main status stays on `ProjectStatus`).

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	main := mkCleanMainRepo(t)
	// Place linked worktree as sibling directory (not under main).
	parent := filepath.Dir(main)
	base := filepath.Base(main)
	linked := filepath.Join(parent, base+"-feat-dirty")
	gitRun(t, main, "worktree", "add", "-b", "feat-dirty", linked)
	// Dirty the linked worktree.
	if err := os.WriteFile(filepath.Join(linked, "dirty.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}
	writeProjectsJSON(t, req.WrkHome, []string{main})
	return nil
}
```
