## Expected

1. Non-zero exit.
2. Combined output mentions missing or not-found remote path (actionable error).
3. No files created under `local-missing/`.

## Side Effects

None — no partial local mirror.

## Errors

- Exit 0 with mirrored files.
- Silent failure without path context.

## Exit Code

Non-zero.

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
	if !strings.Contains(lower, "missing") && !strings.Contains(lower, "not found") && !strings.Contains(lower, "not exist") && !strings.Contains(lower, "no such") {
		t.Fatalf("expected actionable missing-path error; combined:\n%s", resp.Combined)
	}

	assertLocalPathMissing(t, resp.AgentWorkDir, "local-missing")
	assertLocalPathMissing(t, resp.AgentWorkDir, "local-missing/a.txt")
}
```