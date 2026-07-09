## Expected Output

```
dry-run: upload plan
Uploading __LOCAL__/ (2 items, __SIZE__) -> uploads/mirror
Error: upload destination
```

## Expected

1. Non-zero exit.
2. Combined output starts with `dry-run: upload plan` then actionable guard error (destination not empty/missing).
3. `serverHome` file tree unchanged (before/after snapshot match).
4. `uploads/mirror/existing.txt` content unchanged; no incoming files from local tree.

## Side Effects

None — guard fails before simulated upload lines mutate server state.

## Errors

- Exit 0 with mirrored files.
- `existing.txt` overwritten or removed.
- Missing `dry-run: upload plan` banner.

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

	combinedHasAll(t, resp.Combined, "dry-run: upload plan", "uploads/mirror")
	lower := strings.ToLower(resp.Combined)
	if !strings.Contains(lower, "empty") && !strings.Contains(lower, "missing") && !strings.Contains(lower, "not exist") {
		t.Fatalf("expected actionable empty/missing destination error; combined:\n%s", resp.Combined)
	}

	assertTreeSnapshotUnchanged(t, "serverHome", resp.ServerFilesBeforeCLI, resp.ServerFilesAfterCLI)
	assertServerFileContent(t, resp.ServerHome, "uploads/mirror/existing.txt", seedExistingFileContent)
	assertServerPathMissing(t, resp.ServerHome, "uploads/mirror/incoming.txt")
	assertServerPathMissing(t, resp.ServerHome, "uploads/mirror/nested/incoming.txt")
}
```