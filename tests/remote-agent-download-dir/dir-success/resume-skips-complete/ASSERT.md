## Expected Output

```
Downloading uploads/mirror -> ./local-mirror/ (2 items, __SIZE__)
  [1/2] a.txt (__SIZE_A__) — 0% overall
    skipped (already complete, __SIZE_A__ / __SIZE_A__) — __PCT__% overall
  [2/2] sub/b.txt (__SIZE_B__) — __PCT2__% overall
    skipped (already complete, __SIZE_B__ / __SIZE_B__) — 100% overall

Download complete: ./local-mirror/ (2 files, __SIZE__; 2 skipped, 0 resumed)
```

## Expected

1. Exit code 0.
2. Stdout contains `skipped (already complete` for both files before `Download complete`.
3. Summary includes `2 skipped, 0 resumed`.
4. Local files remain correct (no corruption).

## Side Effects

- No redundant byte transfer for complete files.

## Errors

- Files re-downloaded without skip lines.
- Summary omits skip count when skips occurred.

## Exit Code

0.

```go
import (
	"regexp"
	"testing"

	"github.com/xhd2015/doctest/assert"
)

var reSkippedComplete = regexp.MustCompile(`skipped \(already complete`)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}

	assertStdoutEndsWithNewline(t, resp.Stdout)
	combinedHasAll(t, resp.Combined, "skipped", "already complete", "2 skipped", "0 resumed", "Download complete")
	if reSkippedComplete.FindAllString(resp.Stdout, -1) == nil || len(reSkippedComplete.FindAllString(resp.Stdout, -1)) < 2 {
		t.Fatalf("stdout missing two skipped (already complete) lines;\nhave:\n%s", resp.Stdout)
	}

	assertLocalFileContent(t, resp.LocalDir, "a.txt", "alpha\n")
	assertLocalFileContent(t, resp.LocalDir, "sub/b.txt", "bravo\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__SIZE__: type=string
__SIZE_A__: type=string
__SIZE_B__: type=string
__PCT__: type=number
__PCT2__: type=number
---
Downloading uploads/mirror -> ./local-mirror/ (2 items, __SIZE__)
  [1/2] a.txt (__SIZE_A__) — 0% overall
    skipped (already complete, __SIZE_A__ / __SIZE_A__) — __PCT__% overall
  [2/2] sub/b.txt (__SIZE_B__) — __PCT2__% overall
    skipped (already complete, __SIZE_B__ / __SIZE_B__) — 100% overall

Download complete: ./local-mirror/ (2 files, __SIZE__; 2 skipped, 0 resumed)
`)
}
```