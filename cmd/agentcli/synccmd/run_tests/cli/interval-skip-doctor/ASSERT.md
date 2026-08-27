## Expected

1. Harness err nil; `RunErr` empty.
2. Exec called with adjacent `-repeat` `60`.
3. State exitCode 0.

## Side Effects

- State written (fake Exec returns immediately).

## Errors

- None.

## Exit Code

- Nil RunCLI error.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	_ = d
	_ = req
	if err != nil {
		t.Fatalf("harness Run returned unexpected error: %v", err)
	}
	if resp.RunErr != "" {
		t.Fatalf("RunCLI error: %s", resp.RunErr)
	}
	if !resp.ExecCalled {
		t.Fatal("expected Exec on unison run --interval --skip-doctor")
	}
	if !argvHasAdjacent(resp.ExecArgv, "-repeat", "60") {
		t.Fatalf("Exec argv missing adjacent -repeat 60; got %v", resp.ExecArgv)
	}
	if !resp.StateExists || resp.StateExit == nil || *resp.StateExit != 0 {
		t.Fatalf("state exitCode want 0; json=%s", resp.StateJSON)
	}
}
```
