## Expected

- `channels.slack.mode` is `socket`.
- `appToken` and `botToken` use `source=env` with IDs `SLACK_APP_TOKEN` and `SLACK_BOT_TOKEN`.
- Rendered JSON must not contain raw token strings.

```go
import (
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	channels := resp.RenderedJSON["channels"].(map[string]any)
	slack := channels["slack"].(map[string]any)
	if slack["mode"] != "socket" {
		t.Fatalf("mode = %v", slack["mode"])
	}
	app := slack["appToken"].(map[string]any)
	bot := slack["botToken"].(map[string]any)
	if app["source"] != "env" || app["id"] != "SLACK_APP_TOKEN" {
		t.Fatalf("appToken = %v", app)
	}
	if bot["source"] != "env" || bot["id"] != "SLACK_BOT_TOKEN" {
		t.Fatalf("botToken = %v", bot)
	}
	raw := string(resp.RenderedRaw)
	if strings.Contains(raw, "xoxb-render") || strings.Contains(raw, "xapp-render") {
		t.Fatalf("raw tokens leaked in rendered config: %s", raw)
	}
}
```