## Expected

1. `TimeLeft` is exactly `left 2m`.

## Errors

- Wrong unit (hours/days) or off-by-one minute.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.TimeLeft != "left 2m" {
		t.Fatalf("TimeLeft = %q, want %q", resp.TimeLeft, "left 2m")
	}
}
```