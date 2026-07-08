## Expected

1. `ResetDisplay` is exactly `soon` (unchanged raw input).

## Errors

- Empty string, error, or reformatted output.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ResetDisplay != "soon" {
		t.Fatalf("ResetDisplay = %q, want %q", resp.ResetDisplay, "soon")
	}
}
```