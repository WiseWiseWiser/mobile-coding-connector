## Expected

1. History fetch succeeds (no ActionError from history).
2. No history entry has `StartedAt` equal to `2020-01-01T00:00:00Z` (or any year 2020).
3. If any runs remain, they are within the 7-day window relative to test time
   (the seeded recent `2026-07-09T12:00:00Z` may be kept when "now" is 2026-07-10).

## Side Effects

- Old rows removed from persistence and API response.

## Errors

- Ancient run still returned after prune.

## Exit Code

0 from `Run`.

```go
import (
	"strings"
	"testing"
	"time"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ActionError != "" {
		t.Fatalf("history failed: %s", resp.ActionError)
	}
	for _, r := range resp.History {
		if strings.HasPrefix(r.StartedAt, "2020-") {
			t.Fatalf("old run not pruned: %+v", r)
		}
		if r.StartedAt == "2020-01-01T00:00:00Z" {
			t.Fatalf("exact old startedAt still present: %+v", r)
		}
		// Soft check: if parseable, must be within 8d of now (allow clock skew)
		if ts, err := parseRFC3339(r.StartedAt); err == nil {
			if time.Since(ts) > 8*24*time.Hour {
				t.Fatalf("run older than 7d still present: %v", ts)
			}
		}
	}
}
```
