## Expected Output

```
dry-run: download plan
Downloading uploads/mirror -> ./local-mirror/ (__ITEMS__ items, __SIZE__)
...5 lines omitted...
dry-run: download complete: ./local-mirror/ (__FILES__ files, __SIZE__)
```

## Expected

1. Exit code 0.
2. Stdout contains `dry-run: download plan`, `would download`, and `dry-run: download complete`.
3. Stdout ends with `\n` and does not perform real downloads (`Download complete:` without `dry-run:` prefix must not appear).
4. No local files created under `local-mirror/` (before/after snapshot both empty).

## Side Effects

None — no GET download, no local writes.

## Errors

- Local files created under `local-mirror/`.
- Missing `dry-run:` banner or `would download` lines.
- Real `Download complete:` summary (non-dry-run wording).

## Exit Code

0.

```go
import (
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
	combinedHasAll(t, resp.Combined, "dry-run: download plan", "would download", "dry-run: download complete", "local-mirror")
	combinedHasNone(t, resp.Combined, "\nDownload complete:")

	assertTreeSnapshotUnchanged(t, "localDir", resp.LocalFilesBeforeCLI, resp.LocalFilesAfterCLI)
	assertLocalPathMissing(t, resp.LocalDir, "a.txt")
	assertLocalPathMissing(t, resp.LocalDir, "sub/b.txt")

	assert.Output(t, resp.Stdout, `---
version: 2
__ITEMS__: type=number
__FILES__: type=number
__SIZE__: type=string
---
dry-run: download plan
Downloading uploads/mirror -> ./local-mirror/ (__ITEMS__ items, __SIZE__)
...5 lines omitted...
dry-run: download complete: ./local-mirror/ (__FILES__ files, __SIZE__)
`)
}
```