## Expected

1. `AuthHeader` is exactly `Bearer abc`.

## Errors

- Missing space, wrong scheme, or raw token without Bearer.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.AuthHeader != "Bearer abc" {
		t.Fatalf("AuthHeader = %q, want %q", resp.AuthHeader, "Bearer abc")
	}
}
```
