## Expected

1. Exit code 0.
2. Newest `main-*` worktree has non-empty `git branch --show-current`.
3. `git symbolic-ref HEAD` succeeds (not detached).
4. Current branch name equals worktree directory basename (`main-1`).

## Side Effects

Worktree on named branch; remote cleaned.

## Errors

- Detached HEAD in worktree after pull.

## Exit Code

0.

```go
import (
	"os"
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
	wtPath := findWorktreeDirBySuffix(t, base, "main-1")
	if _, err := os.Stat(wtPath); err != nil {
		t.Fatalf("worktree main-1: %v", err)
	}
	assertWorktreeNamedBranch(t, wtPath)
	if strings.TrimSpace(gitPorcelain(t, resp.ProjectDir)) != "" {
		t.Fatalf("remote still dirty")
	}
}
```