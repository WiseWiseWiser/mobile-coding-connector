## Expected Output

```
Uploading __LOCAL__ (__SIZE__) -> uploads/hello.txt
...1 lines omitted...
Upload complete: __REMOTE__ (__SIZE__)
```

## Expected

1. Exit code 0.
2. Stdout contains `Uploading`, `hello.txt`, `Upload complete`, and `uploads/hello.txt`.
3. Stdout ends with `\n` and does **not** contain directory streaming markers (`overall`, `[N/M]` index headers).
4. Remote file `uploads/hello.txt` exists under `serverHome` with fixture bytes.

## Side Effects

- `uploads/hello.txt` created on server with content from `testdata/hello.txt`.

## Errors

- Directory-upload wording on stdout for a single file.
- `overall` rollup suffix or `[N/M]` item headers on single-file chunk lines.
- Missing or wrong remote bytes.

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
	combinedHasAll(t, resp.Combined, "Uploading", "hello.txt", "Upload complete", "uploads/hello.txt")
	combinedHasNone(t, resp.Combined, "files,", "overall", "[1/")

	want := readServerFile(t, resp.ServerHome, "uploads/hello.txt")
	if !strings.Contains(want, "hello from single-file regression") {
		t.Fatalf("remote uploads/hello.txt = %q", want)
	}

	assert.Output(t, resp.Stdout, `---
version: 2
__LOCAL__: type=string
__REMOTE__: type=string
__SIZE__: type=string
---
Uploading __LOCAL__ (__SIZE__) -> uploads/hello.txt
...1 lines omitted...
Upload complete: __REMOTE__ (__SIZE__)
`)
}
```