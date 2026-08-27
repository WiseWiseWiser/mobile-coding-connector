## Expected

1. Harness err nil (error is in `RunErr`).
2. `RunErr` non-empty; mentions `--watch` and `--interactive` (or "cannot").
3. Exec not called.

## Side Effects

- No state write required.

## Errors

- RunCLI returns incompatible-flags error.

## Exit Code

- Non-nil RunCLI error (captured in `RunErr`).

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
		t.Fatal("expected RunCLI error for --watch + --interactive")
	}
	low := strings.ToLower(resp.RunErr)
	if !strings.Contains(low, "--watch") || !strings.Contains(low, "--interactive") {
		t.Fatalf("RunErr should mention both flags; got %q", resp.RunErr)
	}
	if resp.ExecCalled {
		t.Fatal("Exec must not run when --watch and --interactive conflict")
	}
}
```
