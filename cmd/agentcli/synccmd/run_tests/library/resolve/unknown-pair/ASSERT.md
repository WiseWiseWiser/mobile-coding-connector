## Expected

1. Harness err nil.
2. `RunPairErr` contains `unknown pair`.
3. Exec not called.
4. No state file for ghost.

## Side Effects

- None.

## Errors

- Substring: `unknown pair`.

## Exit Code

- Non-nil library error.

```go
import (
	"strings"
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
	if resp.RunPairErr == "" {
		t.Fatal("expected unknown pair error from RunPair")
	}
	if !strings.Contains(resp.RunPairErr, "unknown pair") {
		t.Fatalf("want unknown pair in error; got %q", resp.RunPairErr)
	}
	if resp.ExecCalled {
		t.Fatal("Exec must not run for unknown pair")
	}
	if resp.StateExists {
		t.Fatal("state must not be written for unknown pair")
	}
}
```
