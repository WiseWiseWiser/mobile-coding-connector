## Expected

1. `CanDelete` is `true`.

## Errors

- Delete incorrectly disabled for error tasks.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CanDelete != true {
		t.Fatalf("CanDelete = %v, want true for status %q", resp.CanDelete, req.Status)
	}
}
```
