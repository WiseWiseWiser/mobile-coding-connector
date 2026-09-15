## Expected

1. Status is 409, not 400 or 500.
2. The error names the item and suggests `update`.

## Errors

- Returning 200, or a generic 500 for a duplicate id.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.APIStatus != 409 {
		t.Fatalf("status = %d, body = %s", resp.APIStatus, resp.APIBody)
	}
	if !strings.Contains(resp.APIError, `usage item "grok" already exists`) {
		t.Fatalf("error = %q", resp.APIError)
	}
}
```
