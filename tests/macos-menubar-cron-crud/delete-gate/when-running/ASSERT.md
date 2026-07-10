## Expected

1. `CanDelete` is `false`.

## Errors

- Delete enabled while status is `running`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CanDelete != false {
		t.Fatalf("CanDelete = %v, want false for status %q", resp.CanDelete, req.Status)
	}
}
```
