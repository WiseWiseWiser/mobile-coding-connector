## Expected Output

```
dry-run: upload plan
Uploading __LOCAL__ (__SIZE__) -> uploads/hello.txt
  would upload chunk 1/1 (__SIZE__ / __SIZE__, 100%)
dry-run: upload complete: uploads/hello.txt (__SIZE__)
```

## Expected

1. Exit code 0.
2. Stdout contains `dry-run: upload plan`, `would upload chunk`, and `dry-run: upload complete`.
3. Stdout ends with `\n` and does **not** contain directory streaming markers (`overall`, `[N/M]` index headers).
4. `serverHome` file tree unchanged; `uploads/hello.txt` remains absent.

## Side Effects

None — no upload init/chunk/complete.

## Errors

- Directory-upload wording (`overall`, `[1/`) on single-file dry-run.
- Remote file created on server.
- Real `Upload complete:` summary without `dry-run:` prefix.

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
	combinedHasAll(t, resp.Combined, "dry-run: upload plan", "would upload chunk", "dry-run: upload complete", "uploads/hello.txt")
	combinedHasNone(t, resp.Combined, "files,", "overall", "[1/", "\nUpload complete:")

	assertTreeSnapshotUnchanged(t, "serverHome", resp.ServerFilesBeforeCLI, resp.ServerFilesAfterCLI)
	assertServerPathMissing(t, resp.ServerHome, "uploads/hello.txt")

	assert.Output(t, resp.Stdout, `---
version: 2
__LOCAL__: type=string
__SIZE__: type=string
---
dry-run: upload plan
Uploading __LOCAL__ (__SIZE__) -> uploads/hello.txt
  would upload chunk 1/1 (__SIZE__ / __SIZE__, 100%)
dry-run: upload complete: uploads/hello.txt (__SIZE__)
`)
}
```