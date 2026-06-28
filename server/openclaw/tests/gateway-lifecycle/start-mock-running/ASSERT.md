## Expected

- Start succeeds.
- `Status.Running` and `Status.Mocked` are true.
- `Status.MockPID` is `4242`.
- Generated config file exists with slack socket mode.

```go
import (
	"os"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.StartErr != nil {
		t.Fatalf("Start error: %v", resp.StartErr)
	}
	if !resp.Status.Running || !resp.Status.Mocked || resp.Status.MockPID != 4242 {
		t.Fatalf("status = %+v", resp.Status)
	}
	if _, err := os.Stat(resp.GeneratedPath); err != nil {
		t.Fatalf("generated config missing: %v", err)
	}
	channels, ok := resp.RenderedJSON["channels"].(map[string]any)
	if !ok {
		t.Fatalf("channels missing: %v", resp.RenderedJSON)
	}
	slack, ok := channels["slack"].(map[string]any)
	if !ok || slack["mode"] != "socket" {
		t.Fatalf("slack render = %v", slack)
	}
}
```