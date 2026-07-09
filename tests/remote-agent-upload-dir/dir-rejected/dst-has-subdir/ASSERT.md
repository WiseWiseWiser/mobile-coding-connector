## Expected

1. Non-zero exit.
2. Combined output mentions destination must be missing or empty.
3. `uploads/mirror/child/` still exists and remains empty.
4. No new files written under `uploads/mirror/`.

## Side Effects

None.

## Errors

- Exit 0 because child subdirectory is empty.
- `incoming.txt` or nested uploads appear under `uploads/mirror/`.

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
	if !strings.Contains(lower, "empty") && !strings.Contains(lower, "missing") && !strings.Contains(lower, "not exist") {
		t.Fatalf("expected actionable empty/missing destination error; combined:\n%s", resp.Combined)
	}

	assertServerIsDir(t, resp.ServerHome, "uploads/mirror/child")
	assertServerDirEmpty(t, resp.ServerHome, "uploads/mirror/child")
	assertServerPathMissing(t, resp.ServerHome, "uploads/mirror/incoming.txt")
	assertServerPathMissing(t, resp.ServerHome, "uploads/mirror/nested/incoming.txt")
}
```