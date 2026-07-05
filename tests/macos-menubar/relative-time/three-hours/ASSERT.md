## Expected

1. `TimeLeft` is exactly `left 3h`.

## Errors

- Rounding up to 4h or switching to minutes.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "left 3h" {
		t.Fatalf("TimeLeft = %q, want %q", resp.TimeLeft, "left 3h")
	}
}
```