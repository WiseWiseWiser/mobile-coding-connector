## Expected

1. `TimeLeft` is exactly `left 0m`.

## Errors

- Returning empty or a positive minute count.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "left 0m" {
		t.Fatalf("TimeLeft = %q, want %q", resp.TimeLeft, "left 0m")
	}
}
```