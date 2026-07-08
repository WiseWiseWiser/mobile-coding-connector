## Expected

1. `ResetDisplay` is exactly `Aug 1, 08:00`.

## Errors

- Keeping codex source order `08:00 on 1 Aug` or full month name mismatch.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ResetDisplay != "Aug 1, 08:00" {
		t.Fatalf("ResetDisplay = %q, want %q", resp.ResetDisplay, "Aug 1, 08:00")
	}
}
```