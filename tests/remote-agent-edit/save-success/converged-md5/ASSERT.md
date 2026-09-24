## Expected Output

```
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Remote __REMOTE__ already matches your edits; nothing written
```

## Expected

1. Exit code 0 (the desired content is already on the server).
2. Stdout reports that the remote already matches; no `Saved` line.
3. No conflict recipe (`changed on the server`) and no `Error`.
4. Remote and staged copies both contain `same\n`.

## Side Effects

- None: the write was refused by the precondition, and no merge is needed.

## Errors

- Hard conflict error with a resolution recipe although both sides converged.
- A write happened anyway (precondition bypassed).

## Exit Code

0.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"

	"github.com/xhd2015/doctest/assert"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	assertExit(t, resp, 0)

	combinedHasAll(t, resp.Combined, "already matches your edits; nothing written")
	combinedHasNone(t, resp.Combined, "Saved ", "changed on the server", "Error")

	assertRemoteContent(t, resp, "notes.md", "same\n")
	assertStagedContent(t, resp, "same\n")

	assert.Output(t, resp.Stdout, `---
version: 2
__STAGED__: type=string
__EDITOR__: type=string
__REMOTE__: type=string
---
Downloading notes.md -> __STAGED__ (4 B)
Opening __EDITOR__ __STAGED__
Remote __REMOTE__ already matches your edits; nothing written
`)
}
```
