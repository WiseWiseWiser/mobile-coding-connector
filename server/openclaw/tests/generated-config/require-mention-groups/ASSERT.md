## Expected

- `channels.slack.groups["*"].requireMention` is true.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	channels := resp.RenderedJSON["channels"].(map[string]any)
	slack := channels["slack"].(map[string]any)
	groups := slack["groups"].(map[string]any)
	star := groups["*"].(map[string]any)
	if require, ok := star["requireMention"].(bool); !ok || !require {
		t.Fatalf("groups = %v", groups)
	}
}
```