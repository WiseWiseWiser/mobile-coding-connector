## Expected

1. `AuthHeader` is exactly `Bearer abc`.

## Errors

- Missing space, wrong scheme, or raw token without Bearer.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.AuthHeader != "Bearer abc" {
		t.Fatalf("AuthHeader = %q, want %q", resp.AuthHeader, "Bearer abc")
	}
}
```
