## Expected

1. Harness err nil; `RunErr` non-empty.
2. Error mentions `--interval` and unit hint (`60s` / `unit` / `invalid`).
3. Exec not called.

## Side Effects

- None.

## Errors

- Invalid duration parse.

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
		t.Fatal("expected RunCLI error for bare --interval 60")
	}
	low := strings.ToLower(resp.RunErr)
	if !strings.Contains(low, "--interval") {
		t.Fatalf("RunErr should mention --interval; got %q", resp.RunErr)
	}
	if !strings.Contains(low, "unit") && !strings.Contains(low, "invalid") && !strings.Contains(low, "60s") {
		t.Fatalf("RunErr should hint duration unit; got %q", resp.RunErr)
	}
	if resp.ExecCalled {
		t.Fatal("Exec must not run on invalid --interval")
	}
}
```
