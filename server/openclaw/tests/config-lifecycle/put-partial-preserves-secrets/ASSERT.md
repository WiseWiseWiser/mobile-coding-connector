## Expected

- HTTP 200; response masks tokens as `***`.
- On-disk config keeps `xoxb-keep` and `xapp-keep`.
- `dm_policy` updated to `allowlist`.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.APIStatusCode != 200 {
		t.Fatalf("status = %d body = %s", resp.APIStatusCode, resp.APIBody)
	}
	if !strings.Contains(resp.APIBody, `"bot_token":"***"`) {
		t.Fatalf("response should mask tokens: %s", resp.APIBody)
	}
	if resp.ConfigOnDisk.Slack.BotToken != "xoxb-keep" || resp.ConfigOnDisk.Slack.AppToken != "xapp-keep" {
		t.Fatalf("on-disk secrets dropped: %+v", resp.ConfigOnDisk.Slack)
	}
	if resp.Config.Slack.DMPolicy != "allowlist" {
		t.Fatalf("dm_policy = %q, want allowlist", resp.Config.Slack.DMPolicy)
	}
}
```