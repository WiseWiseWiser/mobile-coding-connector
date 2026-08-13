## Expected

1. `ParseErr` is empty.
2. `WeeklyLimit` is `65%` (percent from progress-bar line under Weekly limit panel).
3. `NextReset` is `August 14, 08:55` (from `Resets:`; bare wall clock, no invented TZ).

## Errors

- Parse failure on new modal labels (`Weekly limit (…)` / `Resets:`) while
  legacy `Weekly limit: N%` / `Next reset:` still work (other leaves).
- Wrong percent or invented PT/UTC suffix.

```go
import (
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ParseErr != "" {
		t.Fatalf("parse error: %s", resp.ParseErr)
	}
	if resp.WeeklyLimit != "65%" {
		t.Fatalf("WeeklyLimit = %q, want 65%%", resp.WeeklyLimit)
	}
	if resp.NextReset != "August 14, 08:55" {
		t.Fatalf("NextReset = %q, want %q", resp.NextReset, "August 14, 08:55")
	}
}
```
