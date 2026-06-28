## Expected

- `gateway_port` is `19002`.
- Slack remains enabled with original tokens on disk.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.Config.GatewayPort != 19002 {
		t.Fatalf("gateway_port = %d, want 19002", resp.Config.GatewayPort)
	}
	if resp.Config.Slack == nil || !resp.Config.Slack.Enabled {
		t.Fatal("slack should remain enabled")
	}
	if resp.ConfigOnDisk.Slack.BotToken != "xoxb-noslack" {
		t.Fatalf("bot token lost: %+v", resp.ConfigOnDisk.Slack)
	}
}
```