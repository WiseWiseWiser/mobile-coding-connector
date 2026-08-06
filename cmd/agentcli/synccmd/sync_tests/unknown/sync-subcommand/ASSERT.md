## Expected

1. `RunErr` non-empty and contains `unknown` (case-sensitive substring).

## Side Effects

- None.

## Errors

- `unknown`.

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
	if resp.RunErr == "" || !strings.Contains(strings.ToLower(resp.RunErr), "unknown") {
		t.Fatalf("want unknown subcommand error; got %q", resp.RunErr)
	}
}
```
