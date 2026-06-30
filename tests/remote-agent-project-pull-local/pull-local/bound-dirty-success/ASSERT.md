## Expected

1. Exit code 0.
2. Summary mentions worktree path under `project-worktrees`.
3. Worktree contains modified `README.md` (`dirty remote`) and `pulled-untracked.txt`.
4. Remote project dir `git status --porcelain` is empty after success.

## Side Effects

Local worktree created; remote hard-reset/cleaned.

## Errors

- Missing files in worktree or dirty remote after pull.

## Exit Code

0.

```go
import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	base := worktreeBaseDir(resp.AgentHome)
	if _, err := os.Stat(base); err != nil {
		t.Fatalf("worktree base missing %s: %v", base, err)
	}
	if !strings.Contains(resp.Combined, "worktree") && !strings.Contains(strings.ToLower(resp.Combined), "pull") {
		t.Fatalf("expected summary in output:\n%s", resp.Combined)
	}

	wtPath := findNewestWorktreeDir(t, base)
	readme, err := os.ReadFile(filepath.Join(wtPath, "README.md"))
	if err != nil {
		t.Fatalf("readme in worktree: %v", err)
	}
	if !strings.Contains(string(readme), "dirty remote") {
		t.Fatalf("worktree README not dirty: %q", readme)
	}
	if _, err := os.Stat(filepath.Join(wtPath, "pulled-untracked.txt")); err != nil {
		t.Fatalf("untracked file missing in worktree: %v", err)
	}

	if porcelain := gitPorcelain(t, resp.ProjectDir); strings.TrimSpace(porcelain) != "" {
		t.Fatalf("remote still dirty:\n%s", porcelain)
	}

	assertWorktreeNamedBranch(t, wtPath)
}

func findNewestWorktreeDir(t *testing.T, base string) string {
	t.Helper()
	var candidates []string
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.Contains(filepath.Base(path), "main-") {
			if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
				candidates = append(candidates, path)
			}
		}
		return nil
	})
	if len(candidates) == 0 {
		t.Fatalf("no main-* worktree under %s", base)
	}
	return candidates[len(candidates)-1]
}
```