## Expected

1. `LaunchErr` is non-nil.
2. Error text mentions `opencode` or `not installed` (case-insensitive).

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
		t.Fatal("expected launch to fail without opencode")
	}
	msg := strings.ToLower(resp.LaunchErr.Error())
	if !strings.Contains(msg, "opencode") && !strings.Contains(msg, "not installed") {
		t.Fatalf("unexpected error: %v", resp.LaunchErr)
	}
}
```