## Expected

1. Exit code 1.
2. Combined output indicates include path is not part of the pull / dirty set.

## Side Effects

Remote remains dirty; no new worktree with pulled state.

## Errors

- Exit 0.

## Exit Code

1.

```go
import (
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
	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "not-in-pull.bin") && !strings.Contains(lower, "include") {
		t.Fatalf("expected include-file / path error;\n%s", resp.Combined)
	}
	if !strings.Contains(lower, "pull") && !strings.Contains(lower, "dirty") && !strings.Contains(lower, "not part") {
		t.Fatalf("expected not-part-of-pull style message;\n%s", resp.Combined)
	}
}
```