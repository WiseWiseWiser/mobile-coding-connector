## Expected

1. `Response.StreamErr` is non-empty.
2. Error indicates missing completion (`done` or similar wording).
3. `Done` is nil.

## Side Effects

None.

## Errors

- Stream returns success without `done` (regression).

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.StreamErr == "" {
		t.Fatal("expected error when stream lacks done frame")
	}
	lower := strings.ToLower(resp.StreamErr)
	if !strings.Contains(lower, "done") && !strings.Contains(lower, "completion") && !strings.Contains(lower, "complete") {
		t.Fatalf("StreamErr = %q, want message about missing completion", resp.StreamErr)
	}
	if resp.Done != nil {
		t.Fatalf("Done = %v, want nil", resp.Done)
	}
}
```
