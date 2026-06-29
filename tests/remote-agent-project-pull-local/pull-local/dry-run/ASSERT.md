## Expected Output

```
<contains>
dry-run
worktree
</contains>
```

## Expected

1. Exit code 0.
2. Combined output describes plan (worktree path, remote dir, or change counts).
3. No `main-*` worktree directory under `project-worktrees`.
4. Remote remains dirty.

## Side Effects

None (no worktree, no remote truncate).

## Errors

- Worktree created or remote cleaned.

## Exit Code

0.

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	assert.Output(t, resp.Combined, `
<contains>
dry
</contains>`)

	base := worktreeBaseDir(resp.AgentHome)
	if err := walkNoMainWorktree(base); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(gitPorcelain(t, resp.ProjectDir)) == "" {
		t.Fatalf("remote should remain dirty after dry-run")
	}
}

func walkNoMainWorktree(base string) error {
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.Contains(filepath.Base(path), "main-") {
			return fmt.Errorf("unexpected worktree dir %s", path)
		}
		return nil
	})
}
```