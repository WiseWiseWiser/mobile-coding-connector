## Expected

1. `StatusLine` is exactly:
   `Cannot reach https://example.com — retry or Test Connection`
2. `StatusContainsToken` is false.
3. Line includes the server host and retry/test guidance.

## Errors

- Omitting host; leaking token; only saying "error".

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Cannot reach https://example.com — retry or Test Connection"
	if resp.StatusLine != want {
		t.Fatalf("StatusLine = %q, want %q", resp.StatusLine, want)
	}
	if resp.StatusContainsToken || strings.Contains(resp.StatusLine, req.Token) {
		t.Fatalf("status leaked token: %q", resp.StatusLine)
	}
	if !strings.Contains(resp.StatusLine, "https://example.com") {
		t.Fatalf("status missing server: %q", resp.StatusLine)
	}
}
```
