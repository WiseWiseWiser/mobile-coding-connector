---
label: slow, negative, e2e
explanation: fake TUI stuck on Update now; expects error / non-ready
---
## Expected

1. `ServiceStatus` is `error` (not `ready`).
2. `ServiceError` is non-empty (timeout or `could not select Skip`).
3. Marker `enter-while-update-now` / `enter-update-now` is **absent**.
4. `MonthlyUsage` is empty.

## Errors

- `status=ready` (must not scrape usage after forcing upgrade or ignoring modal).
- Enter injected while selection is Update now.

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
	for _, name := range resp.MarkerFiles {
		if name == "enter-while-update-now" || name == "enter-update-now" {
			t.Fatalf("production confirmed Update now (marker %q) — must verify Skip before Enter", name)
		}
	}
	if resp.ServiceStatus == "ready" {
		t.Fatalf("status=ready with stuck Update now fake — must not succeed without Skip")
	}
	if resp.ServiceStatus != "error" {
		t.Fatalf("status=%q, want error", resp.ServiceStatus)
	}
	if strings.TrimSpace(resp.ServiceError) == "" {
		t.Fatal("service error empty, want timeout or could not select Skip")
	}
	if strings.TrimSpace(resp.MonthlyUsage) != "" {
		t.Fatalf("monthly_usage=%q, want empty on error", resp.MonthlyUsage)
	}
}
```
