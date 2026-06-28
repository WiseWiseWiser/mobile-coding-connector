## Expected

- `gateway.port` is 18789.
- `agents.defaults.workspace` matches config.
- `agents.defaults.model.primary` matches config.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	gw, ok := resp.RenderedJSON["gateway"].(map[string]any)
	if !ok || int(gw["port"].(float64)) != 18789 {
		t.Fatalf("gateway = %v", resp.RenderedJSON["gateway"])
	}
	agents := resp.RenderedJSON["agents"].(map[string]any)
	defaults := agents["defaults"].(map[string]any)
	if defaults["workspace"] != "~/.openclaw/workspace" {
		t.Fatalf("workspace = %v", defaults["workspace"])
	}
	model := defaults["model"].(map[string]any)
	if model["primary"] != "anthropic/claude-sonnet-4-6" {
		t.Fatalf("model = %v", model)
	}
}
```