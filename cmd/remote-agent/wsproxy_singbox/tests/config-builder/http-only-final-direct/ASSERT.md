## Expected

- `route.final` is `direct`.
- A logical catch-all rule routes HTTP/HTTPS to `web` selector.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("build error: %v", resp.RunErr)
	}
	route, _ := resp.ConfigJSON["route"].(map[string]any)
	if route["final"] != "direct" {
		t.Fatalf("final = %v, want direct", route["final"])
	}
	rules := routeRules(resp.ConfigJSON)
	found := false
	for _, r := range rules {
		if r["type"] == "logical" && r["outbound"] == "web" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing catch-all web rule: %v", rules)
	}
}
```