---
label: slow
explanation: spawns real codex CLI up to 5 times with 90s timeout each
---

## Expected

Every in-process fetch attempt against the **real** codex CLI must succeed:

1. `ResolvedCodexPath` is non-empty (login-shell resolution).
2. `FetchFailureCount` is `0`.
3. `ServiceStatus` is `ready`.
4. `MonthlyUsage` matches `\d+%`.
5. `CreditsUsed` and `CreditsTotal` are non-empty.
6. `NextReset` is non-empty.

## Errors

- `timeout waiting for snapshot frame`
- `timeout waiting for status output`
- `codex not found`
- Any `FetchFailureCount > 0`

```go
import (
	"regexp"
	"strings"
	"testing"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.ResolvedCodexPath == "" {
		t.Fatal("resolved codex path empty")
	}
	if resp.FetchFailureCount != 0 {
		t.Fatalf("fetch failures = %d errors = %v (real codex must complete every attempt)",
			resp.FetchFailureCount, resp.FetchErrors)
	}
	if resp.ServiceStatus != "ready" {
		t.Fatalf("status = %q error = %q, want ready", resp.ServiceStatus, resp.ServiceError)
	}
	if !regexp.MustCompile(`^\d+%$`).MatchString(resp.MonthlyUsage) {
		t.Fatalf("monthly_usage = %q, want N%%", resp.MonthlyUsage)
	}
	if strings.TrimSpace(resp.CreditsUsed) == "" || strings.TrimSpace(resp.CreditsTotal) == "" {
		t.Fatalf("credits_used=%q credits_total=%q, want non-empty", resp.CreditsUsed, resp.CreditsTotal)
	}
	if strings.TrimSpace(resp.NextReset) == "" {
		t.Fatal("next_reset empty")
	}
}
```