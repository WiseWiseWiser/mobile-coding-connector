## Expected

In-process fetch must wait through slow silent boot and return ready usage:

1. `ServiceStatus` is `ready`.
2. `ServiceError` is empty.
3. `MonthlyUsage` is `58%`.
4. `CreditsUsed` is `6,519`.
5. `CreditsTotal` is `11,250`.
6. `NextReset` is `08:00 on 1 Aug`.

## Errors

- `timeout waiting for snapshot frame` (snapshot read gives up before TUI boot completes).
- `timeout waiting for status output`.
- Any non-ready service status.

```go
import "testing"

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ServiceStatus != "ready" {
		t.Fatalf("status = %q error = %q, want ready (bug: snapshot frame timeout during slow boot)",
			resp.ServiceStatus, resp.ServiceError)
	}
	if resp.ServiceError != "" {
		t.Fatalf("service error = %q, want empty", resp.ServiceError)
	}
	if resp.MonthlyUsage != "58%" {
		t.Fatalf("monthly_usage = %q", resp.MonthlyUsage)
	}
	if resp.CreditsUsed != "6,519" {
		t.Fatalf("credits_used = %q", resp.CreditsUsed)
	}
	if resp.CreditsTotal != "11,250" {
		t.Fatalf("credits_total = %q", resp.CreditsTotal)
	}
	if resp.NextReset != "08:00 on 1 Aug" {
		t.Fatalf("next_reset = %q", resp.NextReset)
	}
}
```