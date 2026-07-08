## Expected

1. `ResetDisplay` is exactly `July 9, 20:55`.

## Errors

- Showing Pacific wall clock or wrong offset conversion.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ResetDisplay != "July 9, 20:55" {
		t.Fatalf("ResetDisplay = %q, want %q", resp.ResetDisplay, "July 9, 20:55")
	}
}
```