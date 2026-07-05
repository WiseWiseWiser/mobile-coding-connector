---
label: slow && real-codex
explanation: real codex via tty-watch CLI; wait idle then /status\n\r (~16s)
---

## Expected

Real tty-watch session must reach parseable `/status` output within 30s:

1. `PromptReadySecs` is in `1..20`.
2. `StatusReadySecs` is in `1..25`.
3. `TotalElapsedSecs` is in `1..25`.
4. `StatusFieldsSeen` is true.
5. `MonthlyUsage` matches `\d+%`.

## Errors

- `StatusFieldsSeen=false` (status never rendered).
- `StatusReadySecs > 25` (exceeds observed ~16s budget with margin).

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
	if !resp.StatusFieldsSeen {
		t.Fatalf("status fields never appeared; transcript:\n%s", resp.TTYWatchTranscript)
	}
	if resp.PromptReadySecs < 1 || resp.PromptReadySecs > 20 {
		t.Fatalf("prompt_ready_secs = %d, want 1..20", resp.PromptReadySecs)
	}
	if resp.StatusReadySecs < 1 || resp.StatusReadySecs > 25 {
		t.Fatalf("status_ready_secs = %d, want 1..25; transcript:\n%s",
			resp.StatusReadySecs, resp.TTYWatchTranscript)
	}
	if resp.TotalElapsedSecs < 1 || resp.TotalElapsedSecs > 25 {
		t.Fatalf("total_elapsed_secs = %d, want 1..25", resp.TotalElapsedSecs)
	}
	if !regexp.MustCompile(`^\d+%$`).MatchString(strings.TrimSpace(resp.MonthlyUsage)) {
		t.Fatalf("monthly_usage = %q, want N%%", resp.MonthlyUsage)
	}
}
```