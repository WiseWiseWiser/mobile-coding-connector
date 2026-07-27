## Expected

1. ErrMsg empty.
2. Root has ≥1 child named Grafana, type url, url https://grafana.example.com.
3. Generated id non-empty if not supplied.

## Errors

- Missing child; wrong type/url.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	if resp.ErrMsg != "" {
		t.Fatalf("ErrMsg: %s", resp.ErrMsg)
	}
	n := FindNodeByName(resp.Doc, "Grafana")
	if n == nil {
		t.Fatal("Grafana not found")
	}
	if n.Type != "url" || n.URL != "https://grafana.example.com" {
		t.Fatalf("bad node: %+v", n)
	}
	if n.ID == "" {
		t.Fatal("expected generated non-empty id")
	}
	// must be direct child of root
	found := false
	for _, c := range RootChildren(resp.Doc) {
		if c.Name == "Grafana" {
			found = true
		}
	}
	if !found {
		t.Fatal("Grafana not under root children")
	}
}
```
