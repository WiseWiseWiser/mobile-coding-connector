## Expected

1. The error kind is `invalid`, so the CLI exits non-zero.
2. The message names the item and the provider error.
3. Nothing was registered.

## Errors

- Registering the item anyway, or reporting the failure as a warning.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ErrKind != "invalid" {
		t.Fatalf("error kind = %q, want invalid", resp.ErrKind)
	}
	if resp.Err != "cc-dead fetch failed: Session expired" {
		t.Fatalf("error = %q", resp.Err)
	}
	if len(resp.Added) != 0 {
		t.Fatalf("added = %v, want none", resp.Added)
	}
}
```
