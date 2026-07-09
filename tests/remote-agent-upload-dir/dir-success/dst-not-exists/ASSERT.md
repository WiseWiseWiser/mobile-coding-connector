## Expected Output

```
Uploading __LOCAL__/ (2 items, __SIZE__) -> uploads/mirror
...5 lines omitted...
Upload complete: uploads/mirror (2 files, __SIZE__)
```

## Expected

1. Exit code 0.
2. Stdout reports directory upload with `2 files` and trailing `\n`.
3. `uploads/mirror/a.txt` and `uploads/mirror/sub/b.txt` exist with correct content.

## Side Effects

- Remote tree created under `uploads/mirror/`.

## Errors

- Files land under `uploads/mirror/<basename(localDir>/` instead of `uploads/mirror/`.
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
	combinedHasAll(t, resp.Combined, "Uploading", "2 files", "Upload complete", "uploads/mirror")
	assertServerFileContent(t, resp.ServerHome, "uploads/mirror/a.txt", "alpha\n")
	assertServerFileContent(t, resp.ServerHome, "uploads/mirror/sub/b.txt", "bravo\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__LOCAL__: type=string
__SIZE__: type=string
---
Uploading __LOCAL__/ (2 items, __SIZE__) -> uploads/mirror
...5 lines omitted...
Upload complete: uploads/mirror (2 files, __SIZE__)
`)
}
```