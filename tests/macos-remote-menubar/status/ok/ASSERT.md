## Expected

1. `StatusLine` is exactly `Connected to https://example.com`.
2. `StatusContainsToken` is false (sentinel token must not appear).

## Errors

- Printing token, Bearer header, or localhost keep-alive port.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := "Connected to https://example.com"
	if resp.StatusLine != want {
		t.Fatalf("StatusLine = %q, want %q", resp.StatusLine, want)
	}
	if resp.StatusContainsToken || strings.Contains(resp.StatusLine, req.Token) {
		t.Fatalf("status leaked token: %q", resp.StatusLine)
	}
	if strings.Contains(resp.StatusLine, "127.0.0.1") || strings.Contains(resp.StatusLine, "23312") {
		t.Fatalf("status must not reference local keep-alive: %q", resp.StatusLine)
	}
}
```
