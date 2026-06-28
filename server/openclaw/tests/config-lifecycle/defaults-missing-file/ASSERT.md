## Expected

- `Config.GatewayPort` is `18789`.
- `Config.Slack` is nil or `Enabled` is false.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.Config.GatewayPort != 18789 {
		t.Fatalf("GatewayPort = %d, want 18789", resp.Config.GatewayPort)
	}
	if resp.Config.Slack != nil && resp.Config.Slack.Enabled {
		t.Fatalf("slack should be disabled by default, got %+v", resp.Config.Slack)
	}
}
```