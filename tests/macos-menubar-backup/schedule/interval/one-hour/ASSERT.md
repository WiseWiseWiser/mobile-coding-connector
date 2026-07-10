## Expected

1. `IntervalSeconds` is exactly `3600`.

## Errors

- Different default period without updating this sealed contract.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.IntervalSeconds != 3600 {
		t.Fatalf("IntervalSeconds = %d, want 3600", resp.IntervalSeconds)
	}
}
```
