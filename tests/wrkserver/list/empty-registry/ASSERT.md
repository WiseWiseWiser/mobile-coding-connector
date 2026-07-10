## Expected

1. HTTP status `200`.
2. JSON envelope has `projects` as an empty array (length 0), not a bare array
   and not null-as-missing envelope.

## Errors

- Non-200 status.
- Missing `projects` key or non-empty list.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, resp.Body)
	}
	if resp.Projects == nil {
		// parseResponse leaves nil when key absent; empty array is preferred
		t.Fatalf("projects key missing or null; body=%s", resp.Body)
	}
	if len(resp.Projects) != 0 {
		t.Fatalf("projects len = %d, want 0; body=%s", len(resp.Projects), resp.Body)
	}
}
```
