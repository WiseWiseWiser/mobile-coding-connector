## Expected

- No logical catch-all rule.
- Include suffix `.corp.com` routes to `web`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.RunErr != nil {
		t.Fatalf("build error: %v", resp.RunErr)
	}
	rules := routeRules(resp.ConfigJSON)
	for _, r := range rules {
		if r["type"] == "logical" {
			t.Fatalf("whitelist should not include catch-all: %v", rules)
		}
	}
	found := false
	for _, r := range rules {
		suffixes, _ := r["domain_suffix"].([]any)
		out, _ := r["outbound"].(string)
		for _, s := range suffixes {
			if s == ".corp.com" && out == "web" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("missing .corp.com -> web rule: %v", rules)
	}
}
```