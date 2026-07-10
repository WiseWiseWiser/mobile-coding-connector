## Expected

1. Status **4xx** (validation — path missing). Prefer 400.
2. Non-empty `error` field.

## Errors

- Returning 5xx for missing path (client/validation error).

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 400 || resp.StatusCode > 499 {
		t.Fatalf("status = %d, want 4xx for missing path; body=%s", resp.StatusCode, resp.Body)
	}
	if resp.Error == "" {
		t.Fatalf("want error field; body=%s", resp.Body)
	}
}
```
