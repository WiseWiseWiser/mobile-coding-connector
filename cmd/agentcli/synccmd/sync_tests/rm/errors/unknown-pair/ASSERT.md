## Expected

1. `RunErr` contains `unknown pair`.

## Side Effects

- None.

## Errors

- `unknown pair`.

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
	if resp.RunErr == "" || !strings.Contains(resp.RunErr, "unknown pair") {
		t.Fatalf("want unknown pair; got %q", resp.RunErr)
	}
}
```
