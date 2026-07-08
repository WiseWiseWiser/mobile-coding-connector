## Expected

1. `TimeLeft` is exactly `left 2d5h`.

## Errors

- Wrong day or hour count; single largest unit only.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "left 2d5h" {
		t.Fatalf("TimeLeft = %q, want %q", resp.TimeLeft, "left 2d5h")
	}
}
```