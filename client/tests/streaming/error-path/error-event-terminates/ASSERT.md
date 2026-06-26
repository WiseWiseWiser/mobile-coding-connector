## Expected

1. `Response.StreamErr` is non-empty.
2. Error message contains `upstream_proxy is not configured`.
3. `Done` is nil.

## Side Effects

None.

## Errors

- Stream returns nil error on `error` frame (should fail).

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
		t.Fatal("expected Stream error for error frame, got nil")
	}
	if !strings.Contains(resp.StreamErr, "upstream_proxy is not configured") {
		t.Fatalf("StreamErr = %q, want substring upstream_proxy is not configured", resp.StreamErr)
	}
	if resp.Done != nil {
		t.Fatalf("Done = %v, want nil on error", resp.Done)
	}
}
```
