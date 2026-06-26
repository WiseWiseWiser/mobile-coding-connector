## Expected

1. `Response.ContentType` contains `text/event-stream`.
2. At least one SSE `data:` frame was written.

## Side Effects

None.

## Errors

- `Content-Type` is `application/json` or empty.
- No SSE frames in body.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if !strings.Contains(resp.ContentType, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want substring text/event-stream", resp.ContentType)
	}
	if len(resp.Events) == 0 {
		t.Fatal("expected at least one SSE event in body")
	}
}
```
