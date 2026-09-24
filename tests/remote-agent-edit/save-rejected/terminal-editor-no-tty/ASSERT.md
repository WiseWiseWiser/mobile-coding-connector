---
label: heavy, e2e
---

## Expected Output

stdout:

```
Downloading notes.md -> __STAGED__ (4 B)
```

stderr:

```
Error: editor "vim" needs a terminal on stdin; pass a GUI editor (e.g. --editor=code) or run remote-agent edit from an interactive shell
```

## Expected

1. Exit code 1.
2. The error names the editor and suggests a GUI editor or an interactive shell.
3. The fake `vim` was **not** executed (no marker file).
4. No `Opening` line; remote `notes.md` untouched.

## Side Effects

- Staged copy created; nothing else.

## Errors

- vim launched with piped stdin (hang or garbled terminal).
- Generic exec failure instead of the tty hint.

## Exit Code

1.

```go
import (
	"os"
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExit(t, resp, 1)

	want := `Error: editor "` + resp.EditorPath + `" needs a terminal on stdin; ` +
		`pass a GUI editor (e.g. --editor=code) or run remote-agent edit from an interactive shell`
	combinedHasAll(t, resp.Combined, want)
	combinedHasNone(t, resp.Combined, "Downloading", "Opening ", "Saved ", "Created ")

	if _, statErr := os.Stat(resp.StagedPath); !os.IsNotExist(statErr) {
		t.Fatalf("staged path %s should not exist, stat err = %v", resp.StagedPath, statErr)
	}

	if !strings.Contains(resp.Stderr, "needs a terminal on stdin") {
		t.Fatalf("tty hint should be on stderr;\nstderr:\n%s", resp.Stderr)
	}
	assertEditorDidNotRun(t, resp)
	assertRemoteContent(t, resp, "notes.md", "old\n")
}
```
