## Expected

1. Status is 400.
2. The error names the missing flag and the kind.

## Errors

- Returning 409 or 500 for a field problem.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.APIStatus != 400 {
		t.Fatalf("status = %d, body = %s", resp.APIStatus, resp.APIBody)
	}
	if resp.APIError != "usage item cc: --home is required for kind commandcode" {
		t.Fatalf("error = %q", resp.APIError)
	}
}
```
