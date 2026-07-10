## Expected

1. Status 500–599.
2. Non-empty `error` (may include injected message).
3. Open was called.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 500 || resp.StatusCode > 599 {
		t.Fatalf("status = %d, want 5xx; body=%s", resp.StatusCode, resp.Body)
	}
	if resp.Error == "" {
		t.Fatalf("want error field; body=%s", resp.Body)
	}
	if !resp.OpenCalled {
		t.Fatal("Open should have been called before failure")
	}
	if !strings.Contains(resp.Error, "osascript boom") && !strings.Contains(resp.Body, "osascript boom") {
		t.Logf("error does not echo inject text (ok if wrapped): %q", resp.Error)
	}
}
```
