## Expected Output

```
dry-run: upload plan
Uploading __LOCAL__/ (2 items, __SIZE__) -> uploads/mirror
...5 lines omitted...
dry-run: upload complete: uploads/mirror (2 files, __SIZE__)
```

## Expected

1. Exit code 0.
2. Stdout contains `dry-run: upload plan`, `would upload`, and `dry-run: upload complete`.
3. Stdout ends with `\n` and does not perform real uploads (`Upload complete:` without `dry-run:` prefix must not appear).
4. `serverHome` file tree unchanged (before/after snapshot match).
5. `uploads/mirror/a.txt` and `uploads/mirror/sub/b.txt` remain absent.

## Side Effects

None — no server mkdir, upload init/chunk/complete.

## Errors

- Remote files created under `uploads/mirror/`.
- Missing `dry-run:` banner or `would upload` chunk lines.
- Real `Upload complete:` summary (non-dry-run wording).

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
	combinedHasAll(t, resp.Combined, "dry-run: upload plan", "would upload", "dry-run: upload complete", "uploads/mirror")
	combinedHasNone(t, resp.Combined, "\nUpload complete:")

	assertTreeSnapshotUnchanged(t, "serverHome", resp.ServerFilesBeforeCLI, resp.ServerFilesAfterCLI)
	assertServerPathMissing(t, resp.ServerHome, "uploads/mirror/a.txt")
	assertServerPathMissing(t, resp.ServerHome, "uploads/mirror/sub/b.txt")

	assert.Output(t, resp.Stdout, `---
version: 2
__LOCAL__: type=string
__SIZE__: type=string
---
dry-run: upload plan
Uploading __LOCAL__/ (2 items, __SIZE__) -> uploads/mirror
...5 lines omitted...
dry-run: upload complete: uploads/mirror (2 files, __SIZE__)
`)
}
```