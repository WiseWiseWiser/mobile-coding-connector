## Expected Output

```
dry-run: download plan
Downloading uploads/mirror -> ./local-mirror/ (3 items, __SIZE__)
  [1/3] a.txt (__SIZE_A__) — 0% overall
    would skip (already complete, __SIZE_A__ / __SIZE_A__) — __PCT__% overall
  [2/3] sub/b.txt (__SIZE_B__) — __PCT2__% overall
    would skip (already complete, __SIZE_B__ / __SIZE_B__) — __PCT3__% overall
  [3/3] big.bin (__SIZE_BIN__) — __PCT4__% overall
    would resume at __HALF__ / __SIZE_BIN__ (50%) — __PCT5__% overall
    would download chunk ... — __PCT6__% overall

dry-run: download complete: ./local-mirror/ (3 files, __SIZE__; 2 would skip, 1 would resume)
```

## Expected

1. Exit code 0.
2. Stdout contains `would skip (already complete`, `would resume at`, and summary `2 would skip, 1 would resume`.
3. Local file bytes unchanged (before/after snapshot match).
4. `big.bin` remains 512 bytes (not fully downloaded).

## Side Effects

None — no GET download, no local append.

## Errors

- Local `big.bin` grows to 1024 bytes.
- Missing `would skip` or `would resume` preview lines.
- Real `downloaded`/`skipped`/`resumed at` lines (non-dry-run wording).

## Exit Code

0.

```go
import (
	"os"
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
	combinedHasAll(t, resp.Combined,
		"dry-run: download plan",
		"would skip",
		"already complete",
		"would resume at",
		"would download",
		"2 would skip",
		"1 would resume",
		"dry-run: download complete",
	)
	combinedHasNone(t, resp.Combined, "\nDownload complete:", "downloaded ", "skipped (already complete", "resumed at ")

	assertTreeSnapshotUnchanged(t, "localDir", resp.LocalFilesBeforeCLI, resp.LocalFilesAfterCLI)

	binPath := localFilePath(resp.LocalDir, "big.bin")
	info, err := os.Stat(binPath)
	if err != nil {
		t.Fatalf("stat %s: %v", binPath, err)
	}
	if info.Size() != resumePreviewPrefill {
		t.Fatalf("big.bin size = %d, want %d (dry-run must not append)", info.Size(), resumePreviewPrefill)
	}

	assertLocalFileContent(t, resp.LocalDir, "a.txt", "alpha\n")
	assertLocalFileContent(t, resp.LocalDir, "sub/b.txt", "bravo\n")

	resumeIdx := strings.Index(resp.Stdout, "would resume at")
	completeIdx := strings.Index(resp.Stdout, "dry-run: download complete")
	if resumeIdx < 0 || completeIdx < 0 || resumeIdx > completeIdx {
		t.Fatalf("would resume at line must appear before dry-run complete;\nhave:\n%s", resp.Stdout)
	}

	assert.Output(t, resp.Stdout, `---
version: 2
__SIZE__: type=string
__SIZE_A__: type=string
__SIZE_B__: type=string
__SIZE_BIN__: type=string
__HALF__: type=string
__PCT__: type=number
__PCT2__: type=number
__PCT3__: type=number
__PCT4__: type=number
__PCT5__: type=number
__PCT6__: type=number
---
dry-run: download plan
Downloading uploads/mirror -> ./local-mirror/ (3 items, __SIZE__)
  [1/3] a.txt (__SIZE_A__) — 0% overall
    would skip (already complete, __SIZE_A__ / __SIZE_A__) — __PCT__% overall
  [2/3] sub/b.txt (__SIZE_B__) — __PCT2__% overall
    would skip (already complete, __SIZE_B__ / __SIZE_B__) — __PCT3__% overall
  [3/3] big.bin (__SIZE_BIN__) — __PCT4__% overall
    would resume at __HALF__ / __SIZE_BIN__ (50%) — __PCT5__% overall
    would download chunk ... — __PCT6__% overall

dry-run: download complete: ./local-mirror/ (3 files, __SIZE__; 2 would skip, 1 would resume)
`)
}
```