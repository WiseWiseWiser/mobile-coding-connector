## Expected Output

```
Downloading uploads/hello.txt -> ./hello.txt
...1 lines omitted...
Download complete: __LOCAL__ (__SIZE__)
```

## Expected

1. Exit code 0.
2. Stdout contains `Downloading`, `hello.txt`, `Download complete`, and a `downloaded` progress line.
3. Stdout ends with `\n` and does **not** contain directory streaming markers (`overall`, `[N/M]` index headers).
4. Local file `./hello.txt` exists under `agentWorkDir` with fixture bytes.

## Side Effects

- `hello.txt` created in `agentWorkDir` with content from `testdata/hello.txt`.

## Errors

- Directory-download wording on stdout for a single file.
- `overall` rollup suffix or `[N/M]` item headers on single-file lines.
- Missing or wrong local bytes.

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
	combinedHasAll(t, resp.Combined, "Downloading", "hello.txt", "downloaded", "Download complete")
	combinedHasNone(t, resp.Combined, "files,", "overall", "[1/", "items")

	localFile := localFilePath(resp.AgentWorkDir, "hello.txt")
	data, err := os.ReadFile(localFile)
	if err != nil {
		t.Fatalf("read local hello.txt: %v", err)
	}
	if !strings.Contains(string(data), "hello from single-file regression") {
		t.Fatalf("local hello.txt = %q", string(data))
	}

	assert.Output(t, resp.Stdout, `---
version: 2
__LOCAL__: type=string
__SIZE__: type=string
---
Downloading uploads/hello.txt -> ./hello.txt
...1 lines omitted...
Download complete: __LOCAL__ (__SIZE__)
`)
}
```