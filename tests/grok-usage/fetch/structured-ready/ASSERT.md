## Expected

1. `ServiceStatus` is `ready`.
2. `WeeklyLimit` is `61%`.
3. `NextReset` is `July 17, 08:55` (raw back-compat, bare local).
4. `ResetAt` is non-empty and parses as RFC3339.
5. `ResetDisplay` is non-empty (UI token; bare local → same wall clock text).
6. `TimeLeft` matches `^left ` (menubar unit policy prefix).

## Errors

- Ready without structured fields (empty `reset_at` / `reset_display` / `time_left`).
- Inventing a PT/UTC suffix on raw `next_reset`.

```go
import (
	"regexp"
	"testing"
	"time"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ServiceStatus != "ready" {
		t.Fatalf("status = %q, want ready", resp.ServiceStatus)
	}
	if resp.WeeklyLimit != "61%" {
		t.Fatalf("weekly_limit = %q, want 61%%", resp.WeeklyLimit)
	}
	if resp.NextReset != "July 17, 08:55" {
		t.Fatalf("next_reset = %q, want %q", resp.NextReset, "July 17, 08:55")
	}
	if resp.ResetAt == "" {
		t.Fatal("reset_at empty; want RFC3339 absolute instant")
	}
	if _, parseErr := time.Parse(time.RFC3339, resp.ResetAt); parseErr != nil {
		t.Fatalf("reset_at not RFC3339: %q (%v)", resp.ResetAt, parseErr)
	}
	if resp.ResetDisplay == "" {
		t.Fatal("reset_display empty; want non-empty UI token for Reset {…}")
	}
	if resp.TimeLeft == "" {
		t.Fatal("time_left empty; want countdown starting with \"left \"")
	}
	if ok, _ := regexp.MatchString(`^left `, resp.TimeLeft); !ok {
		t.Fatalf("time_left = %q, want prefix \"left \"", resp.TimeLeft)
	}
}
```
