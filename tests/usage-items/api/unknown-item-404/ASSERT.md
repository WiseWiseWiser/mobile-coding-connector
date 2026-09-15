## Expected

1. Status is 404.
2. The error lists the known ids so the user can correct the typo.

## Errors

- Returning 400 for a missing item, or omitting the known ids.

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
	if resp.APIStatus != 404 {
		t.Fatalf("status = %d, body = %s", resp.APIStatus, resp.APIBody)
	}
	if !strings.Contains(resp.APIError, `unknown usage item "missing"`) {
		t.Fatalf("error = %q", resp.APIError)
	}
	if !strings.Contains(resp.APIError, "known items: grok, codex") {
		t.Fatalf("error must list known ids: %q", resp.APIError)
	}
}
```
