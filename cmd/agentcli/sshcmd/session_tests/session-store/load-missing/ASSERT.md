## Expected

1. `LoadErr` is empty (missing is not an error).
2. `Loaded` is nil.

## Side Effects

- None (no file created).

## Errors

None expected from the store.

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
	if resp.LoadErr != "" {
		t.Fatalf("Load missing must not error; got %q", resp.LoadErr)
	}
	if resp.Loaded != nil {
		t.Fatalf("Loaded must be nil when file missing; got %+v", resp.Loaded)
	}
}
```
