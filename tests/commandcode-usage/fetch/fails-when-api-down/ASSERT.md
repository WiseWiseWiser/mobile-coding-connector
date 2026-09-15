## Expected

1. Status is `error`.
2. The message is the provider's network error.

## Errors

- Reporting `not authenticated` or an empty message when the API cannot be reached.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "error" {
		t.Fatalf("status = %q", resp.Status)
	}
	if resp.Error != "Network error: unable to reach API" {
		t.Fatalf("error = %q", resp.Error)
	}
}
```
