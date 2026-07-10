## Expected

1. HTTP status in 4xx range.
2. JSON body has non-empty `error` string.

## Errors

- Treating non-git path as success.
- 5xx without structured error body.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 400 || resp.StatusCode >= 500 {
		t.Fatalf("status = %d, want 4xx; body=%s", resp.StatusCode, resp.Body)
	}
	if resp.Error == "" {
		t.Fatalf("expected non-empty error field; body=%s", resp.Body)
	}
}
```
