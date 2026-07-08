## Expected

1. `TimeLeft` is exactly `left 1m`.

## Errors

- Returning `left 2m` (rounding up) or `left 0m`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "left 1m" {
		t.Fatalf("TimeLeft = %q, want %q", resp.TimeLeft, "left 1m")
	}
}
```