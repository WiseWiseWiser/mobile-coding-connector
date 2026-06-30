## Expected

- HTTP 200.
- Top-level keys: `sessions`, `page`, `page_size`, `total`, `total_pages`.
- Each session object has: `id`, `name`, `cwd`, `created_at`, `status`, `connected`.

```go
import (
	"net/http"
	"testing"
)

var listTopKeys = []string{"sessions", "page", "page_size", "total", "total_pages"}
var sessionKeys = []string{"id", "name", "cwd", "created_at", "status", "connected"}

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ListStatus != http.StatusOK {
		t.Fatalf("status %d", resp.ListStatus)
	}
	for _, k := range listTopKeys {
		if _, ok := resp.ListJSON[k]; !ok {
			t.Fatalf("missing top-level key %q in %v", k, resp.ListJSON)
		}
	}
	sessions, ok := resp.ListJSON["sessions"].([]any)
	if !ok {
		t.Fatalf("sessions is not array: %T", resp.ListJSON["sessions"])
	}
	if len(sessions) == 0 {
		t.Fatal("expected at least one seeded session")
	}
	first, ok := sessions[0].(map[string]any)
	if !ok {
		t.Fatal("session entry is not object")
	}
	for _, k := range sessionKeys {
		if _, ok := first[k]; !ok {
			t.Fatalf("missing session key %q in %v", k, first)
		}
	}
}
```