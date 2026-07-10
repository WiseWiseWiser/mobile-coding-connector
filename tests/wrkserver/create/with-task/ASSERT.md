## Expected

1. HTTP status `200`.
2. `path` under `$WRK_HOME/worktrees/`.
3. Both `path` and `branch` contain slug `fix-login`.

## Side Effects

- Worktree directory exists at `path`.

## Errors

- Missing slug in path or branch.
- Path not under worktrees root.

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
	if !strings.HasPrefix(resp.Path, worktreesRoot) {
		t.Fatalf("path %q not under %q", resp.Path, worktreesRoot)
	}
	if !strings.Contains(resp.Path, "fix-login") {
		t.Fatalf("path missing fix-login slug: %q", resp.Path)
	}
	if !strings.Contains(resp.Branch, "fix-login") {
		t.Fatalf("branch missing fix-login slug: %q", resp.Branch)
	}
	if st, err := os.Stat(resp.Path); err != nil || !st.IsDir() {
		t.Fatalf("created path not a directory: %q err=%v", resp.Path, err)
	}
}
```
