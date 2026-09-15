## Expected

1. Status is 400.
2. The error is the missing-id message, not `unknown usage item ""`.

## Errors

- Returning 404 for an empty id.

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
	if resp.APIError != "usage item: --id is required" {
		t.Fatalf("error = %q", resp.APIError)
	}
}
```
