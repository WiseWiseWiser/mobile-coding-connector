## Expected

1. `TimeLeft` is exactly `left 1min`.

## Errors

- Returning `left 2min` (rounding up) or `left 0min`.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "left 1min" {
		t.Fatalf("TimeLeft = %q, want %q", resp.TimeLeft, "left 1min")
	}
}
```