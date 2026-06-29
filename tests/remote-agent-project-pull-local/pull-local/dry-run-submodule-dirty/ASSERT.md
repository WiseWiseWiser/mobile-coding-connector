## Expected

1. Exit code 1.
2. Submodule error appears; output must not read like a successful dry-run plan (no worktree path allocation).
3. No `main-*` worktree under agent home.

## Side Effects

None.

## Errors

- Exit 0 or worktree created.

## Exit Code

1.

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
	if resp.ExitCode == 0 {
		t.Fatalf("expected failure; combined:\n%s", resp.Combined)
	}
	combined := strings.ToLower(resp.Combined)
	if !strings.Contains(combined, "submod") {
		t.Fatalf("expected submodule error;\n%s", resp.Combined)
	}
	base := worktreeBaseDir(resp.AgentHome)
	filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if d != nil && d.IsDir() && strings.Contains(filepath.Base(path), "main-") {
			t.Fatalf("unexpected worktree %s on failed dry-run", path)
		}
		return nil
	})
}
```