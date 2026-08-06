## Expected

1. Harness err nil.
2. `RunPairErr` non-empty; prefer tokens `run` / `name` / `require`.
3. Exec not called.

## Side Effects

- None required.

## Errors

- Non-empty missing-name error.

## Exit Code

- Non-nil.

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
		t.Fatal("expected error when RunPair name is empty")
	}
	low := strings.ToLower(resp.RunPairErr)
	ok := strings.Contains(low, "run") || strings.Contains(low, "name") || strings.Contains(low, "require")
	if !ok {
		t.Fatalf("error should mention run/name/require; got %q", resp.RunPairErr)
	}
	if resp.ExecCalled {
		t.Fatal("Exec must not run when name missing")
	}
}
```
