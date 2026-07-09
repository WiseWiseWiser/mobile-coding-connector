## Expected

1. `StatusLine` is exactly:
   `Token rejected — open Configure… to update credentials`
2. `StatusContainsConfig` is true.
3. `StatusContainsToken` is false.

## Errors

- Printing the rejected token; silent failure.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	want := statusUnauthorized
	if resp.StatusLine != want {
		t.Fatalf("StatusLine = %q, want %q", resp.StatusLine, want)
	}
	if !resp.StatusContainsConfig {
		t.Fatal("expected StatusLine to mention Configure")
	}
	if resp.StatusContainsToken || strings.Contains(resp.StatusLine, req.Token) {
		t.Fatalf("status leaked token: %q", resp.StatusLine)
	}
}
```
