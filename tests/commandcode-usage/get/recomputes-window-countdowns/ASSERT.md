## Expected

1. The first read reports `left 2h` and `left 2d6h`.
2. One hour later the same cache reports `left 1h` and `left 2d5h`.

## Errors

- Returning a stale countdown, or refetching to get a fresh one.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.FiveHourLeft != "left 2h" || resp.WeeklyLeft != "left 2d6h" {
		t.Fatalf("first read = %q / %q", resp.FiveHourLeft, resp.WeeklyLeft)
	}
	if resp.FiveHourLeftAfter != "left 1h" || resp.WeeklyLeftAfter != "left 2d5h" {
		t.Fatalf("second read = %q / %q", resp.FiveHourLeftAfter, resp.WeeklyLeftAfter)
	}
}
```
