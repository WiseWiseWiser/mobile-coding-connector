## Expected

- Loaded config retains `xoxb-roundtrip` and `xapp-roundtrip`.
- On-disk JSON retains plaintext tokens (not masked).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.Config.Slack == nil || resp.Config.Slack.BotToken != "xoxb-roundtrip" {
		t.Fatalf("loaded bot token = %+v", resp.Config.Slack)
	}
	if resp.ConfigOnDisk.Slack.BotToken != "xoxb-roundtrip" || resp.ConfigOnDisk.Slack.AppToken != "xapp-roundtrip" {
		t.Fatalf("on-disk tokens not preserved: %+v", resp.ConfigOnDisk.Slack)
	}
}
```