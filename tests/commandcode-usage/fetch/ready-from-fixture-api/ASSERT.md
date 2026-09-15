## Expected

1. Status is `ready` with no error.
2. Plan is `GOAT` and `active`.
3. Cycle is 5% used, `$66.19` left, 1,870 requests, 26 days to renewal.
4. The 5-hour window is 15% and the weekly window 11%.
5. The usage URL points at the account's settings page.
6. The detail panel is plain text and starts at the plan headline.

## Errors

- Reporting the wrong plan, dropping the request count, or leaking ANSI codes into the panel.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ready" {
		t.Fatalf("status = %q, error = %q", resp.Status, resp.Error)
	}
	if resp.PlanName != "GOAT" || resp.PlanStatus != "active" {
		t.Fatalf("plan = %q/%q", resp.PlanName, resp.PlanStatus)
	}
	if resp.CyclePercent != 5 || resp.CycleRemaining != "$66.19" {
		t.Fatalf("cycle = %d%% / %q", resp.CyclePercent, resp.CycleRemaining)
	}
	if resp.CycleRequests != "1,870" || resp.DaysToRenewal != 26 {
		t.Fatalf("requests = %q, days = %d", resp.CycleRequests, resp.DaysToRenewal)
	}
	if resp.FiveHourPercent != 15 || resp.WeeklyPercent != 11 {
		t.Fatalf("windows = %d/%d", resp.FiveHourPercent, resp.WeeklyPercent)
	}
	if resp.UsageURL != "https://commandcode.ai/acct/settings/usage" {
		t.Fatalf("usage url = %q", resp.UsageURL)
	}
	if !strings.Contains(resp.Detail, "GOAT Plan") || !strings.Contains(resp.Detail, "5% used") {
		t.Fatalf("detail panel = %q", resp.Detail)
	}
	if strings.Contains(resp.Detail, "\x1b[") {
		t.Fatalf("detail panel must be plain text: %q", resp.Detail)
	}
	if resp.UpdatedAt == "" {
		t.Fatal("a ready response must stamp updated_at")
	}
}
```
