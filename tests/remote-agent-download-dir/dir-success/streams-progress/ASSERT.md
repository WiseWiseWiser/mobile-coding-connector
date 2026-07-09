## Expected Output

```
Downloading uploads/stream-mirror -> ./local-stream/ (2 items, __SIZE__)
  [1/2] a.txt (__SIZE_A__) — __PCT__% overall
    downloaded __SIZE_A__ / __SIZE_A__ (100%) — __PCT2__% overall
  [2/2] sub/b.txt (__SIZE_B__) — __PCT3__% overall
    downloaded __SIZE_B__ / __SIZE_B__ (100%) — 100% overall

Download complete: ./local-stream/ (2 files, __SIZE__)
```

## Expected

1. Exit code 0.
2. Stdout streams hierarchical progress before the summary: at least one `[N/M]` item header, at least one `overall` rollup suffix, and at least one indented `downloaded` line — all appearing before `Download complete`.
3. Banner uses `items` count (2 items); summary uses `2 files`.
4. Local paths `local-stream/a.txt` and `local-stream/sub/b.txt` exist with correct content.

## Side Effects

- Mirrored tree under `local-stream/` with streaming stdout.

## Errors

- Silent gap between banner and `Download complete` (no incremental progress).
- Missing `[N/M]` index or `overall` suffix on directory download lines.
- Single-file-style flat lines without directory context.

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
	reItemIndex      = regexp.MustCompile(`\[[1-9][0-9]*/[1-9][0-9]*\]`)
	reDownloadedLine = regexp.MustCompile(`(?m)^    downloaded `)
	reOverallLine    = regexp.MustCompile(`(?m) — [0-9]+% overall`)
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
	combinedHasAll(t, resp.Combined, "Downloading", "items", "overall", "downloaded", "Download complete", "local-stream")
	combinedHasNone(t, resp.Combined, "created ")

	assertLocalFileContent(t, resp.LocalDir, "a.txt", "alpha\n")
	assertLocalFileContent(t, resp.LocalDir, "sub/b.txt", "bravo\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__SIZE__: type=string
__SIZE_A__: type=string
__SIZE_B__: type=string
__PCT__: type=number
__PCT2__: type=number
__PCT3__: type=number
---
Downloading uploads/stream-mirror -> ./local-stream/ (2 items, __SIZE__)
  [1/2] a.txt (__SIZE_A__) — __PCT__% overall
    downloaded __SIZE_A__ / __SIZE_A__ (100%) — __PCT2__% overall
  [2/2] sub/b.txt (__SIZE_B__) — __PCT3__% overall
    downloaded __SIZE_B__ / __SIZE_B__ (100%) — 100% overall

Download complete: ./local-stream/ (2 files, __SIZE__)
`)
}

func assertStreamingProgressBeforeComplete(t *testing.T, stdout string) {
	t.Helper()
	completeIdx := strings.Index(stdout, "Download complete")
	if completeIdx < 0 {
		t.Fatal("stdout missing Download complete")
	}
	prefix := stdout[:completeIdx]

	if !reItemIndex.MatchString(prefix) {
		t.Fatalf("stdout before summary missing [N/M] item header;\nprefix:\n%s", prefix)
	}
	if !reOverallLine.MatchString(prefix) {
		t.Fatalf("stdout before summary missing overall rollup suffix;\nprefix:\n%s", prefix)
	}
	if !reDownloadedLine.MatchString(prefix) {
		t.Fatalf("stdout before summary missing indented downloaded progress line;\nprefix:\n%s", prefix)
	}
}
```