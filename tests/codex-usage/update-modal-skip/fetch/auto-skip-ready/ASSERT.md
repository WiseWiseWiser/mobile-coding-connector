## Expected

In-process fetch must auto-Skip the update menu and return ready usage:

1. `ServiceStatus` is `ready`.
2. `ServiceError` is empty.
3. `MonthlyUsage` is `58%`.
4. `CreditsUsed` is `6,519`.
5. `CreditsTotal` is `11,250`.
6. `NextReset` is `08:00 on 1 Aug`.
7. Marker `enter-update-now` is **absent** (must not confirm Update now).

## Errors

- `timeout waiting for status output` (no auto-Skip / banner still blocks).
- `status=error` for any reason.
- Silent upgrade path (`enter-update-now` marker).

```go
import (
	"path/filepath"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range resp.MarkerFiles {
		if name == "enter-update-now" {
			t.Fatalf("production Entered while Update now selected (marker %s) — verify-before-Enter required",
				filepath.Join(req.MarkerDir, name))
		}
	}
	if resp.ServiceStatus != "ready" {
		t.Fatalf("status=%q error=%q, want ready after auto-Skip (bug: hang on update modal/banner)",
			resp.ServiceStatus, resp.ServiceError)
	}
	if strings.TrimSpace(resp.ServiceError) != "" {
		t.Fatalf("service error=%q, want empty", resp.ServiceError)
	}
	if resp.MonthlyUsage != "58%" {
		t.Fatalf("monthly_usage=%q, want 58%%", resp.MonthlyUsage)
	}
	if resp.CreditsUsed != "6,519" {
		t.Fatalf("credits_used=%q, want 6,519", resp.CreditsUsed)
	}
	if resp.CreditsTotal != "11,250" {
		t.Fatalf("credits_total=%q, want 11,250", resp.CreditsTotal)
	}
	if resp.NextReset != "08:00 on 1 Aug" {
		t.Fatalf("next_reset=%q, want 08:00 on 1 Aug", resp.NextReset)
	}
}
```
