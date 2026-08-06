## Expected

1. `RunErr` non-empty.

## Side Effects

- None.

## Errors

- Non-empty.

## Exit Code

- Non-nil error.

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
		t.Fatal("expected error for rm without name")
	}
}
```
