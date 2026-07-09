## Expected Output

```
Downloading uploads/mirror -> ./local-mirror/ (1 items, __SIZE__)
  [1/1] big.bin (__SIZE__) — 0% overall
    resumed at __HALF__ / __SIZE__ (50%) — __PCT__% overall
    downloaded __SIZE__ / __SIZE__ (100%) — 100% overall

Download complete: ./local-mirror/ (1 files, __SIZE__; 0 skipped, 1 resumed)
```

## Expected

1. Exit code 0.
2. Stdout contains `resumed at` with half-file offset before completion line.
3. Summary includes `0 skipped, 1 resumed`.
4. Local `big.bin` is 1024 bytes matching remote content.

## Side Effects

- Partial file appended from byte 512 onward.

## Errors

- Full re-download without `resumed at` line.
- Truncated or corrupted final bytes.

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
	combinedHasAll(t, resp.Combined, "resumed at", "1 resumed", "Download complete", "big.bin")

	want := string(repeatBytePattern(partialFileSize, 42))
	assertLocalFileContent(t, resp.LocalDir, "big.bin", want)

	resumedIdx := strings.Index(resp.Stdout, "resumed at")
	completeIdx := strings.Index(resp.Stdout, "Download complete")
	if resumedIdx < 0 || completeIdx < 0 || resumedIdx > completeIdx {
		t.Fatalf("resumed at line must appear before Download complete;\nhave:\n%s", resp.Stdout)
	}

	assert.Output(t, resp.Stdout, `---
version: 2
__SIZE__: type=string
__HALF__: type=string
__PCT__: type=number
---
Downloading uploads/mirror -> ./local-mirror/ (1 items, __SIZE__)
  [1/1] big.bin (__SIZE__) — 0% overall
    resumed at __HALF__ / __SIZE__ (50%) — __PCT__% overall
    downloaded __SIZE__ / __SIZE__ (100%) — 100% overall

Download complete: ./local-mirror/ (1 files, __SIZE__; 0 skipped, 1 resumed)
`)
}
```