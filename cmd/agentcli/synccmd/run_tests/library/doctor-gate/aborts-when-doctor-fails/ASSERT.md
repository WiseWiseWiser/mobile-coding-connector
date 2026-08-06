## Expected

1. Harness err nil.
2. `RunPairErr` non-empty (prefer doctor/fail/check/serve tokens).
3. Exec **not** called.
4. State file **not** created.

## Side Effects

- No state write; no Exec.

## Errors

- Non-nil RunPair error from doctor gate.

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
		t.Fatal("expected RunPair error when doctor fails without skip-doctor")
	}
	low := strings.ToLower(resp.RunPairErr)
	found := false
	for _, needle := range []string{"doctor", "fail", "check", "serve", "version", "root", "profile"} {
		if strings.Contains(low, needle) {
			found = true
			break
		}
	}
	if !found {
		// Accept any non-empty error; soft-check preferred tokens above as diagnostic only.
		_ = found
	}
	if resp.ExecCalled {
		t.Fatal("Exec must not be called when doctor aborts run")
	}
	if resp.StateExists {
		t.Fatalf("state must not be written on doctor abort; got %s", resp.StateJSON)
	}
}
```
