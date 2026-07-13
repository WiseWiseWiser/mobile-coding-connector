## Expected

1. `ParseErr` is empty.
2. `WeeklyLimit` is `6%`.
3. `NextReset` is `July 9, 16:55 UTC` (UTC preserved, not rewritten to PT).

## Errors

- Parse failure or timezone rewritten incorrectly.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr != "" {
		t.Fatalf("parse error: %s", resp.ParseErr)
	}
	if resp.WeeklyLimit != "6%" {
		t.Fatalf("WeeklyLimit = %q, want 6%%", resp.WeeklyLimit)
	}
	if resp.NextReset != "July 9, 16:55 UTC" {
		t.Fatalf("NextReset = %q, want %q", resp.NextReset, "July 9, 16:55 UTC")
	}
}
```
