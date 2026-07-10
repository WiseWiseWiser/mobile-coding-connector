## Expected

1. HTTP status `200` (not 4xx for empty task).
2. `path` under `$WRK_HOME/worktrees/`.
3. `path` and `branch` do not contain a residual empty-slug artifact that would
   appear as a dangling `-` double segment from whitespace; specifically no
   `fix-login` and branch has no trailing `-` only from blank slug.

## Errors

- 4xx rejecting whitespace task (should normalize to empty).
- Spurious slug segments from untrimmed task.

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
		t.Fatalf("status = %d, want 200 (whitespace task = no slug); body=%s", resp.StatusCode, resp.Body)
	}
	if resp.Path == "" || resp.Branch == "" {
		t.Fatalf("path/branch empty: path=%q branch=%q body=%s", resp.Path, resp.Branch, resp.Body)
	}
	worktreesRoot := filepath.Join(req.WrkHome, "worktrees")
	if !strings.HasPrefix(resp.Path, worktreesRoot) {
		t.Fatalf("path %q not under %q", resp.Path, worktreesRoot)
	}
	// Same as no-task: no fix-login; branch should not end with lone hyphen from blank slug.
	if strings.HasSuffix(resp.Branch, "-") || strings.Contains(resp.Branch, "--") {
		t.Fatalf("branch looks like blank-slug artifact: %q", resp.Branch)
	}
	if st, err := os.Stat(resp.Path); err != nil || !st.IsDir() {
		t.Fatalf("created path not a directory: %q err=%v", resp.Path, err)
	}
}
```
