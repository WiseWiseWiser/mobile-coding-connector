## Expected

1. `ServiceStatus` is `ready`.
2. `MonthlyUsage` is `58%`.
3. `NextReset` is `08:00 on 1 Aug` (raw back-compat).
4. `ResetAt` is non-empty and parses as RFC3339.
5. `ResetDisplay` is non-empty (UI token for `Reset {…}`).
6. `TimeLeft` matches `^left ` (menubar unit policy prefix).

## Errors

- Ready without structured fields (empty `reset_at` / `reset_display` / `time_left`).

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
	if resp.MonthlyUsage != "58%" {
		t.Fatalf("monthly_usage = %q, want 58%%", resp.MonthlyUsage)
	}
	if resp.NextReset != "08:00 on 1 Aug" {
		t.Fatalf("next_reset = %q, want %q", resp.NextReset, "08:00 on 1 Aug")
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
