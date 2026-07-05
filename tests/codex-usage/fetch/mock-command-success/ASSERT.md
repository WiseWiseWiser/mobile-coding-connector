## Expected

1. `ServiceStatus` is `ready`.
2. `MonthlyUsage` is `58%`.
3. `CreditsUsed` is `6,519`.
4. `CreditsTotal` is `11,250`.
5. `NextReset` is `08:00 on 1 Aug`.
6. `UpdatedAt` is non-empty RFC3339 timestamp.

## Errors

- Service reports loading/error or missing fields.

```go
import (
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
	if resp.UpdatedAt == "" {
		t.Fatal("updated_at empty")
	}
	if _, parseErr := time.Parse(time.RFC3339, resp.UpdatedAt); parseErr != nil {
		t.Fatalf("updated_at not RFC3339: %v", parseErr)
	}
}
```