## Expected

1. `AuthHeader` is empty (not `Bearer` or `Bearer `).

## Errors

- Emitting a bare `Bearer` prefix with no credentials.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.AuthHeader != "" {
		t.Fatalf("AuthHeader = %q, want empty", resp.AuthHeader)
	}
}
```
