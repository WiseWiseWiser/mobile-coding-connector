## Expected

1. HTTP status `200`.
2. `path` is non-empty and under `$WRK_HOME/worktrees/`.
3. `branch` is non-empty.
4. Neither `path` nor `branch` contains task slug `fix-login`.

## Side Effects

- New git worktree directory exists at `path`.

## Errors

- Path outside WrkHome worktrees root.
- Unexpected task slug in names.

```go
import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, resp.Body)
	}
	if resp.Path == "" || resp.Branch == "" {
		t.Fatalf("path/branch empty: path=%q branch=%q body=%s", resp.Path, resp.Branch, resp.Body)
	}
	worktreesRoot := filepath.Join(req.WrkHome, "worktrees")
	if !strings.HasPrefix(resp.Path, worktreesRoot+string(os.PathSeparator)) && resp.Path != worktreesRoot {
		// also accept path under worktrees with trailing variations
		if !strings.HasPrefix(resp.Path, worktreesRoot) {
			t.Fatalf("path %q not under %q", resp.Path, worktreesRoot)
		}
	}
	if strings.Contains(resp.Path, "fix-login") || strings.Contains(resp.Branch, "fix-login") {
		t.Fatalf("unexpected task slug in path/branch: path=%q branch=%q", resp.Path, resp.Branch)
	}
	if st, err := os.Stat(resp.Path); err != nil || !st.IsDir() {
		t.Fatalf("created path not a directory: %q err=%v", resp.Path, err)
	}
}
```
