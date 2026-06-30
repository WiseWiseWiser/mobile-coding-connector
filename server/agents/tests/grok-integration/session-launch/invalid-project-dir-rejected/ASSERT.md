## Expected

1. `LaunchErr` is non-nil.
2. Error mentions `project` or `directory` (case-insensitive).

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
		t.Fatal("expected error for invalid project dir")
	}
	msg := strings.ToLower(resp.LaunchErr.Error())
	if !strings.Contains(msg, "project") && !strings.Contains(msg, "directory") {
		t.Fatalf("error: %v", resp.LaunchErr)
	}
}
```