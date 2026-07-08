## Expected

1. `TimeLeft` is exactly `left 3d4h`.

## Errors

- Single-unit `left 3d` only, or wrong hour count.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "left 3d4h" {
		t.Fatalf("TimeLeft = %q, want %q", resp.TimeLeft, "left 3d4h")
	}
}
```