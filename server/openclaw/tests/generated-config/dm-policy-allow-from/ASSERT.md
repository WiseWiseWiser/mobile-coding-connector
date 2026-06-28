## Expected

- `channels.slack.dmPolicy` is `allowlist`.
- `channels.slack.allowFrom` contains `U123` and `U456`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	channels := resp.RenderedJSON["channels"].(map[string]any)
	slack := channels["slack"].(map[string]any)
	if slack["dmPolicy"] != "allowlist" {
		t.Fatalf("dmPolicy = %v", slack["dmPolicy"])
	}
	allow, ok := slack["allowFrom"].([]any)
	if !ok || len(allow) != 2 {
		t.Fatalf("allowFrom = %v", slack["allowFrom"])
	}
}
```