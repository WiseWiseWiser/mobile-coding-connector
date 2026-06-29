## Expected

1. Final exit code 0.
2. `InvocationCount` is 2.
3. Both `main-1` and `main-2` worktree directories exist under `project-worktrees`.

## Side Effects

Two worktrees; remote clean after second pull.

## Errors

- Missing `main-2` suffix directory.

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
	if resp.InvocationCount != 2 {
		t.Fatalf("invocation count=%d want 2", resp.InvocationCount)
	}
	base := worktreeBaseDir(resp.AgentHome)
	has1 := worktreeSuffixExists(base, "main-1")
	has2 := worktreeSuffixExists(base, "main-2")
	if !has1 || !has2 {
		t.Fatalf("main-1=%v main-2=%v under %s", has1, has2, base)
	}
	if strings.TrimSpace(gitPorcelain(t, resp.ProjectDir)) != "" {
		t.Fatalf("remote should be clean after second pull")
	}
}

func worktreeSuffixExists(base, suffix string) bool {
	found := false
	_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasSuffix(filepath.Base(path), suffix) {
			found = true
		}
		return nil
	})
	return found
}
```