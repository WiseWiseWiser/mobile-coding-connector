## Expected

1. Harness err nil.
2. `RunErr` empty.
3. Both LocalEnsure and RemoteEnsure called (default both).

## Side Effects

- Both ensure hooks invoked via CLI path.

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
	if !resp.LocalEnsureCalled {
		t.Fatal("expected LocalEnsure for default install (both)")
	}
	if !resp.RemoteEnsureCalled {
		t.Fatal("expected RemoteEnsure for default install (both)")
	}
}
```
