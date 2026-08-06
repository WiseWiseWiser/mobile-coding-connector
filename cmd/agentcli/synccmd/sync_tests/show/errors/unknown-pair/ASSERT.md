## Expected

1. `RunErr` contains `unknown pair`.

## Side Effects

- None.

## Errors

- Substring: `unknown pair`.

## Exit Code

- Non-nil error.

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
		t.Fatal("expected unknown pair error")
	}
	if !strings.Contains(resp.RunErr, "unknown pair") {
		t.Fatalf("want unknown pair in error; got %q", resp.RunErr)
	}
}
```
