## Expected

1. `CanDelete` is `false`.

## Errors

- Delete enabled while status is `running`.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CanDelete != false {
		t.Fatalf("CanDelete = %v, want false for status %q", resp.CanDelete, req.Status)
	}
}
```
