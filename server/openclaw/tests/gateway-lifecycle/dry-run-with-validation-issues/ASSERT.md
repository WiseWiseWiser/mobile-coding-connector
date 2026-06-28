## Expected

- `DryRun.Mocked` is true.
- `DryRun.Issues` contains bot token validation message.
- Gateway is not running.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !resp.DryRun.Mocked {
		t.Fatal("dry run should report mocked=true")
	}
	found := false
	for _, issue := range resp.DryRun.Issues {
		if strings.Contains(issue, "bot token") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("issues missing bot token error: %v", resp.DryRun.Issues)
	}
	if resp.State.Running {
		t.Fatal("dry-run must not start gateway")
	}
}
```