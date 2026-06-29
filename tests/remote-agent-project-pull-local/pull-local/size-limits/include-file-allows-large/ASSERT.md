## Expected

1. Exit code 0.
2. Worktree contains `big.bin` with size ≥ 2 MiB.
3. Remote porcelain empty after default truncate.

## Side Effects

Worktree under `project-worktrees`; remote cleaned.

## Errors

- Missing `big.bin` in worktree or exit non-zero.

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
	wtPath := findNewestWorktreeDirSize(t, base)
	bigPath := filepath.Join(wtPath, bigUntrackedRel)
	info, err := os.Stat(bigPath)
	if err != nil {
		t.Fatalf("big.bin in worktree: %v", err)
	}
	if info.Size() < 2*perFileCapBytes {
		t.Fatalf("big.bin size %d want >= %d", info.Size(), 2*perFileCapBytes)
	}
	if porcelain := gitPorcelain(t, resp.ProjectDir); strings.TrimSpace(porcelain) != "" {
		t.Fatalf("remote still dirty:\n%s", porcelain)
	}
}

func findNewestWorktreeDirSize(t *testing.T, base string) string {
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