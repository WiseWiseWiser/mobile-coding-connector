## Expected

1. `ParseErr` is empty.
2. `WeeklyLimit` is `59%`.
3. `NextReset` is `July 17, 08:55` (bare wall clock = local time when TZ omitted).

## Errors

- Parse failure (no-TZ line not matched) or invented PT/UTC suffix.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr != "" {
		t.Fatalf("parse error: %s", resp.ParseErr)
	}
	if resp.WeeklyLimit != "59%" {
		t.Fatalf("WeeklyLimit = %q, want 59%%", resp.WeeklyLimit)
	}
	if resp.NextReset != "July 17, 08:55" {
		t.Fatalf("NextReset = %q, want %q", resp.NextReset, "July 17, 08:55")
	}
}
```
