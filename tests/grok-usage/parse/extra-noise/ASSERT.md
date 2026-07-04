## Expected

1. `ParseErr` is empty.
2. `WeeklyLimit` is `12%`.
3. `NextReset` is `August 1, 09:30 PT`.

## Errors

- Parser confused by surrounding noise.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr != "" {
		t.Fatalf("parse error: %s", resp.ParseErr)
	}
	if resp.WeeklyLimit != "12%" {
		t.Fatalf("WeeklyLimit = %q, want 12%%", resp.WeeklyLimit)
	}
	if resp.NextReset != "August 1, 09:30 PT" {
		t.Fatalf("NextReset = %q", resp.NextReset)
	}
}
```