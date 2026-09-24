## Expected Output

stderr:

```
Error: editor "definitely-not-an-editor-xyz" not found in PATH; install it or pass --editor=<command>
```

## Expected

1. Exit code 1.
2. The error names the editor and suggests `--editor=<command>`.
3. Nothing was staged: no `Downloading` line, no staged copy, no `Opening` line.
4. Remote `notes.md` still contains `old\n`.

## Side Effects

- None on either side (the run fails before staging).

## Errors

- Empty or cryptic exec error instead of a hint.
- A wasted download before the editor is validated.

## Exit Code

1.

```go
import (
	"os"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExit(t, resp, 1)

	combinedHasAll(t, resp.Combined,
		`Error: editor "definitely-not-an-editor-xyz" not found in PATH; install it or pass --editor=<command>`,
	)
	combinedHasNone(t, resp.Combined, "Downloading", "Opening ", "Saved ", "Created ")

	if _, statErr := os.Stat(resp.StagedPath); !os.IsNotExist(statErr) {
		t.Fatalf("staged path %s should not exist, stat err = %v", resp.StagedPath, statErr)
	}
	assertRemoteContent(t, resp, "notes.md", "old\n")
}
```
