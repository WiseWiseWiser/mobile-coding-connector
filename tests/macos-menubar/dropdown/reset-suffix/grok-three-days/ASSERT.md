## Expected

1. `ResetSuffix` is exactly `, left 3d`.

## Errors

- Missing leading comma/space or wrong relative unit.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ResetSuffix != ", left 3d" {
		t.Fatalf("ResetSuffix = %q, want %q", resp.ResetSuffix, ", left 3d")
	}
}
```