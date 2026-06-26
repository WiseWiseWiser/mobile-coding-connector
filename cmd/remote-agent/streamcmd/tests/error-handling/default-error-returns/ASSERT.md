## Expected

1. `RunErr` is non-empty.
2. Error contains `doctor failed`.
3. `AfterCalled` is false (stream aborted before done).

## Side Effects

None.

## Errors

- `Run` returns nil on `error` SSE frame.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run returned unexpected error: %v", err)
	}
	if resp.RunErr == "" {
		t.Fatal("expected RunErr for error SSE event")
	}
	if !strings.Contains(resp.RunErr, "doctor failed") {
		t.Fatalf("RunErr = %q, want substring doctor failed", resp.RunErr)
	}
	if resp.AfterCalled {
		t.Fatal("After must not run when stream ends with error")
	}
}
```
