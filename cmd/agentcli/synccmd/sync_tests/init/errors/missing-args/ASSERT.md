## Expected

1. `RunErr` non-empty.
2. No pair `mad-max` in store (store missing or without that name).

## Side Effects

- Must not create a successful pair entry for incomplete args.

## Errors

- Error string non-empty (mentions init/require/usage ideally).

## Exit Code

- Non-nil library error.

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
	if resp.RunErr == "" {
		t.Fatal("expected RunCLI error for missing init args")
	}
	if p := pairByName(resp, "mad-max"); p != nil {
		t.Fatalf("pair must not be created on missing args; got %+v", p)
	}
}
```
