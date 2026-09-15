## Expected

1. The error kind is `invalid`.
2. The message names the missing flag and the kind.

## Errors

- Accepting the item, or reporting a generic error without the flag name.

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
	if resp.Err != "usage item cc: --home is required for kind commandcode" {
		t.Fatalf("error = %q", resp.Err)
	}
}
```
