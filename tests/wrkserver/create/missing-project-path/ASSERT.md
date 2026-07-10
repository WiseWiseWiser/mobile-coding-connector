## Expected

1. HTTP status in 4xx range.
2. JSON body has non-empty `error` string.

## Errors

- 2xx success without project_path.
- Non-JSON or empty error message.

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
