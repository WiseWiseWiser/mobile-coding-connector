## Expected

1. Harness err nil.
2. `RunErr` empty.
3. Exec called.
4. Exec argv contains adjacent `-repeat` `watch`.
5. State exists with exitCode 0.

## Side Effects

- `{StoreDir}/state/mad-max.json` written (fake Exec returns immediately).

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
		t.Fatal("expected Exec via CLIOpts.Exec on unison run --watch --skip-doctor")
	}
	if !argvHasAdjacent(resp.ExecArgv, "-repeat", "watch") {
		t.Fatalf("Exec argv missing adjacent -repeat watch; got %v", resp.ExecArgv)
	}
	if !resp.StateExists {
		t.Fatalf("expected state file at %s", resp.StatePath)
	}
	if resp.StateExit == nil || *resp.StateExit != 0 {
		t.Fatalf("state exitCode want 0; json=%s", resp.StateJSON)
	}
}
```
