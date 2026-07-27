## Expected

1. `CanDelete` is `true`.

## Errors

- Delete incorrectly disabled for idle tasks.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CanDelete != true {
		t.Fatalf("CanDelete = %v, want true for status %q", resp.CanDelete, req.Status)
	}
}
```
