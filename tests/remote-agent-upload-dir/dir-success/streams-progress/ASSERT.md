## Expected Output

```
Uploading __LOCAL__/ (2 items, __SIZE__) -> uploads/stream-mirror
  [1/2] a.txt (__SIZE_A__) — __PCT__% overall
    chunk 1/1 uploaded (__SIZE_A__ / __SIZE_A__, 100%) — __PCT2__% overall
  [2/2] sub/b.txt (__SIZE_B__) — __PCT3__% overall
    chunk 1/1 uploaded (__SIZE_B__ / __SIZE_B__, 100%) — 100% overall

Upload complete: uploads/stream-mirror (2 files, __SIZE__)
```

## Expected

1. Exit code 0.
2. Stdout streams hierarchical progress before the summary: at least one `[N/M]` item header, at least one `overall` rollup suffix, and at least one indented `chunk` line — all appearing before `Upload complete`.
3. Banner uses `items` count (files only here: 2 items); summary uses `2 files`.
4. Remote paths `uploads/stream-mirror/a.txt` and `uploads/stream-mirror/sub/b.txt` exist with correct content.

## Side Effects

- Mirrored tree under `uploads/stream-mirror/` with streaming stdout.

## Errors

- Silent gap between banner and `Upload complete` (no incremental progress).
- Missing `[N/M]` index or `overall` suffix on directory upload lines.
- Single-file-style flat chunk lines without directory context.

## Exit Code

0.

```go
import (
	"regexp"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var (
	reItemIndex   = regexp.MustCompile(`\[[1-9][0-9]*/[1-9][0-9]*\]`)
	reChunkLine   = regexp.MustCompile(`(?m)^    chunk `)
	reOverallLine = regexp.MustCompile(`(?m) — [0-9]+% overall`)
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
	assertStreamingProgressBeforeComplete(t, resp.Stdout)
	combinedHasAll(t, resp.Combined, "Uploading", "items", "overall", "chunk", "Upload complete", "uploads/stream-mirror")
	combinedHasNone(t, resp.Combined, "created ")

	assertServerFileContent(t, resp.ServerHome, "uploads/stream-mirror/a.txt", "alpha\n")
	assertServerFileContent(t, resp.ServerHome, "uploads/stream-mirror/sub/b.txt", "bravo\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__LOCAL__: type=string
__SIZE__: type=string
__SIZE_A__: type=string
__SIZE_B__: type=string
__PCT__: type=number
__PCT2__: type=number
__PCT3__: type=number
---
Uploading __LOCAL__/ (2 items, __SIZE__) -> uploads/stream-mirror
  [1/2] a.txt (__SIZE_A__) — __PCT__% overall
    chunk 1/1 uploaded (__SIZE_A__ / __SIZE_A__, 100%) — __PCT2__% overall
  [2/2] sub/b.txt (__SIZE_B__) — __PCT3__% overall
    chunk 1/1 uploaded (__SIZE_B__ / __SIZE_B__, 100%) — 100% overall

Upload complete: uploads/stream-mirror (2 files, __SIZE__)
`)
}

func assertStreamingProgressBeforeComplete(t *testing.T, stdout string) {
	t.Helper()
	completeIdx := strings.Index(stdout, "Upload complete")
	if completeIdx < 0 {
		t.Fatal("stdout missing Upload complete")
	}
	prefix := stdout[:completeIdx]

	if !reItemIndex.MatchString(prefix) {
		t.Fatalf("stdout before summary missing [N/M] item header;\nprefix:\n%s", prefix)
	}
	if !reOverallLine.MatchString(prefix) {
		t.Fatalf("stdout before summary missing overall rollup suffix;\nprefix:\n%s", prefix)
	}
	if !reChunkLine.MatchString(prefix) {
		t.Fatalf("stdout before summary missing indented chunk progress line;\nprefix:\n%s", prefix)
	}
}
```