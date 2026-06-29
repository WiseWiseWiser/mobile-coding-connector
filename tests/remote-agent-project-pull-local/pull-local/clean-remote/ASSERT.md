## Expected

1. Exit code 1.
2. Combined output indicates nothing to pull (clean worktree / no changes).

## Side Effects

No worktree directory created under `project-worktrees`.

## Errors

- Exit 0.

## Exit Code

1.

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
	if resp.ExitCode == 0 {
		t.Fatalf("expected failure; combined:\n%s", resp.Combined)
	}
	combined := strings.ToLower(resp.Combined)
	if !strings.Contains(combined, "pull") && !strings.Contains(combined, "clean") && !strings.Contains(combined, "nothing") {
		t.Fatalf("expected nothing-to-pull style message;\n%s", resp.Combined)
	}
	base := worktreeBaseDir(resp.AgentHome)
	if _, err := os.Stat(base); err == nil {
		// directory may exist from other runs; ensure no main-* leaf
		if hasWorktreeSuffix(base) {
			t.Fatalf("unexpected worktree created for clean remote under %s", base)
		}
	}
}

func hasWorktreeSuffix(base string) bool {
	entries, _ := os.ReadDir(base)
	return len(entries) > 0
}
```