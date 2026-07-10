## Expected

1. HTTP status `200`.
2. Empty projects list.
3. Notably **not** 404 — demonstrates host-chosen base works.

## Errors

- Only `/api/wrk/projects` works while `/custom/projects` 404s (hardcoded base).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode == 404 {
		t.Fatalf("GET /custom/projects returned 404 — base must be host-owned via Register; body=%s", resp.Body)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, resp.Body)
	}
	if len(resp.Projects) != 0 {
		t.Fatalf("projects len = %d, want 0; body=%s", len(resp.Projects), resp.Body)
	}
}
```
