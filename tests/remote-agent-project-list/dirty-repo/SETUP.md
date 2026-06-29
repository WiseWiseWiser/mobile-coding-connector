# Scenario

**Feature**: dirty worktree shows per-type porcelain counts

```
# commit baseline files, then untracked + modified + renamed + deleted
remote-agent project list -> Worktree: dirty (1 added, 1 changed, 1 renamed, 1 deleted)
```

## Preconditions

Git repo with one committed baseline.

## Steps

1. `git init` on `main`; configure test user.
2. Create and commit three tracked files: `tracked.txt`, `to-delete.txt`, `to-rename.txt`
   (commit message `Initial commit`).
3. **Added (1)**: create `untracked.txt` (not staged).
4. **Changed (1)**: overwrite `tracked.txt` on disk.
5. **Renamed (1)**: `git mv to-rename.txt renamed.txt`.
6. **Deleted (1)**: remove `to-delete.txt` from disk.
7. Register project `dirty-repo-test` (`dirty-001`).

## Context

Porcelain line count = 4. Untracked files count as **added** per requirement.
Exact expected summary: `dirty (1 added, 1 changed, 1 renamed, 1 deleted)`.

```go
import (
	"os"
	"path/filepath"
	"testing"
)

func Setup(t *testing.T, req *Request) error {
	dir := mkProjectDir(t)
	gitInitWithMain(t, dir)

	for name, content := range map[string]string{
		"tracked.txt":  "baseline tracked\n",
		"to-delete.txt": "will delete\n",
		"to-rename.txt": "will rename\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			return err
		}
	}
	gitRun(t, dir, "add", "tracked.txt", "to-delete.txt", "to-rename.txt")
	gitRun(t, dir, "commit", "-m", "Initial commit")

	// 1 added (untracked)
	if err := os.WriteFile(filepath.Join(dir, "untracked.txt"), []byte("new file\n"), 0644); err != nil {
		return err
	}
	// 1 changed
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("modified tracked\n"), 0644); err != nil {
		return err
	}
	// 1 renamed
	gitRun(t, dir, "mv", "to-rename.txt", "renamed.txt")
	// 1 deleted
	if err := os.Remove(filepath.Join(dir, "to-delete.txt")); err != nil {
		return err
	}

	req.Project = ProjectEntry{
		ID:   "dirty-001",
		Name: "dirty-repo-test",
		Dir:  dir,
	}
	return nil
}
```