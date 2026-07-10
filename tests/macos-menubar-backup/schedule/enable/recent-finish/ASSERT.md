## Expected

1. `ShouldRun` is `false` (enable only; no immediate run).

## Errors

- Kicking off a redundant backup inside the 1h window.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ShouldRun {
		t.Fatal("ShouldRun = true, want false when last finish ≤ 1h ago")
	}
}
```
