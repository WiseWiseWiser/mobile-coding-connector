## Expected

1. `LaunchErr` is non-nil.
2. Error contains `unknown` (case-insensitive).

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.LaunchErr == nil {
		t.Fatal("expected error for unknown agent")
	}
	if !strings.Contains(strings.ToLower(resp.LaunchErr.Error()), "unknown") {
		t.Fatalf("error: %v", resp.LaunchErr)
	}
}
```