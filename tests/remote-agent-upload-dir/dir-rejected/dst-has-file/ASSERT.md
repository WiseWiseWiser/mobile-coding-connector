---
label: heavy, e2e
explanation: "L3 smoke: product binary upload-dir path"
---

## Expected

1. Non-zero exit.
2. Combined output mentions destination must be missing or empty (actionable guard message).
3. `uploads/mirror/existing.txt` content unchanged.
4. No uploaded files from local tree appear under `uploads/mirror/`.

## Side Effects

None — no partial directory upload.

## Errors

- Exit 0 with mirrored files.
- `existing.txt` overwritten or removed.

## Exit Code

Non-zero.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
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

	assertServerFileContent(t, resp.ServerHome, "uploads/mirror/existing.txt", seedExistingFileContent)
	assertServerPathMissing(t, resp.ServerHome, "uploads/mirror/incoming.txt")
	assertServerPathMissing(t, resp.ServerHome, "uploads/mirror/nested/incoming.txt")
}
```