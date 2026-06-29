## Expected

1. Exit code 1.
2. Combined output mentions the 1 MB (or 1MB) per-file limit and `--include-file`.

## Side Effects

No successful worktree with `big.bin` content; remote stays dirty.

## Errors

- Exit 0 or missing size-limit guidance.

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
	if !strings.Contains(lower, "include-file") && !strings.Contains(lower, "include file") {
		t.Fatalf("expected --include-file hint;\n%s", resp.Combined)
	}
	if !strings.Contains(lower, "1 mb") && !strings.Contains(lower, "1mb") && !strings.Contains(lower, "1048576") {
		t.Fatalf("expected 1 MB limit mention;\n%s", resp.Combined)
	}
}
```