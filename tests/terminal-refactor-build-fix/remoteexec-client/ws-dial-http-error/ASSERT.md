## Expected

- WS dial against non-upgrading handler returns error.
- Error text includes HTTP 401 (or `Unauthorized`) and JSON body snippet.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if resp.WSError == "" {
		t.Fatal("expected dial error text in resp.WSError")
	}
	errText := resp.WSError
	if !strings.Contains(errText, "401") && !strings.Contains(errText, "Unauthorized") {
		t.Fatalf("error %q missing HTTP 401 status", errText)
	}
	if !strings.Contains(errText, "unauthorized") {
		t.Fatalf("error %q missing body snippet from %q", errText, req.WSDialHTTPBody)
	}
}
```