## Expected Output

```
Downloading uploads/mirror -> ./local-mirror/ (__ITEMS__ items, __SIZE__)
...5 lines omitted...
Download complete: ./local-mirror/ (__FILES__ files, __SIZE__)
```

## Expected

1. Exit code 0.
2. Stdout reports directory download with `2 files` and trailing `\n`.
3. Local paths `local-mirror/a.txt` and `local-mirror/sub/b.txt` exist with correct content.

## Side Effects

- Local tree created under `local-mirror/`.

## Errors

- Files land under `local-mirror/<basename>/` instead of `local-mirror/`.
- Missing `sub/b.txt`.

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
	combinedHasAll(t, resp.Combined, "Downloading", "2 files", "Download complete", "local-mirror")
	assertLocalFileContent(t, resp.LocalDir, "a.txt", "alpha\n")
	assertLocalFileContent(t, resp.LocalDir, "sub/b.txt", "bravo\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__ITEMS__: type=number
__FILES__: type=number
__SIZE__: type=string
---
Downloading uploads/mirror -> ./local-mirror/ (__ITEMS__ items, __SIZE__)
...5 lines omitted...
Download complete: ./local-mirror/ (__FILES__ files, __SIZE__)
`)
}
```