## Expected

1. HTTP status `200` (route registered; not 404).
2. Empty `projects` list.

## Errors

- 404 — leaf not mounted under `/api/wrk`.
- Hardcoded different prefix ignoring `Register` base.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode == 404 {
		t.Fatalf("GET /api/wrk/projects returned 404 — Register did not mount leaf; body=%s", resp.Body)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, resp.Body)
	}
	if len(resp.Projects) != 0 {
		t.Fatalf("projects len = %d, want 0; body=%s", len(resp.Projects), resp.Body)
	}
}
```
