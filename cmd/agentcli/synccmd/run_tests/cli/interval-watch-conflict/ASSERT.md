## Expected

1. Harness err nil; `RunErr` non-empty.
2. Error mentions `--interval` and `--watch`.
3. Exec not called.

## Side Effects

- None required.

## Errors

- Incompatible continuous-mode flags.

## Exit Code

- Non-nil RunCLI error in `RunErr`.

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
	if resp.RunErr == "" {
		t.Fatal("expected RunCLI error for --interval + --watch")
	}
	low := strings.ToLower(resp.RunErr)
	if !strings.Contains(low, "--interval") || !strings.Contains(low, "--watch") {
		t.Fatalf("RunErr should mention both flags; got %q", resp.RunErr)
	}
	if resp.ExecCalled {
		t.Fatal("Exec must not run when --interval and --watch conflict")
	}
}
```
