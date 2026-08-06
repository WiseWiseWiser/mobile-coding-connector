## Expected

1. Harness err nil.
2. `RunPairErr` empty.
3. Exec called despite serve down.
4. State exists with exitCode 0.

## Side Effects

- State written after Exec.

## Errors

- None.

## Exit Code

- Nil.

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
	if resp.RunPairErr != "" {
		t.Fatalf("RunPair should succeed with skip-doctor; err=%s", resp.RunPairErr)
	}
	if !resp.ExecCalled {
		t.Fatal("expected Exec when SkipDoctor=true even if serve down")
	}
	if !resp.StateExists {
		t.Fatalf("expected state file at %s", resp.StatePath)
	}
	if resp.StateExit == nil || *resp.StateExit != 0 {
		t.Fatalf("state exitCode want 0; json=%s", resp.StateJSON)
	}
}
```
