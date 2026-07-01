## Expected

- `RunInteractive` returns a non-nil error containing `boom`.

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
		t.Fatal("expected client error text in resp.WSError")
	}
	if !strings.Contains(resp.WSError, req.WSErrorMessage) {
		t.Fatalf("error %q should contain %q", resp.WSError, req.WSErrorMessage)
	}
}
```