---
label: slow && negative
explanation: stuck update menu; waits for error/timeout (~90s service ctx until early Skip failure)
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
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
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
