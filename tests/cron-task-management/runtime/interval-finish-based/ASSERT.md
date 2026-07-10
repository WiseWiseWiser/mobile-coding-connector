## Expected

1. Create succeeds.
2. At least 2 runs in `History` (or `RunCount >= 2`), OR `MarkerLines >= 2`.
3. If `LastFinishedAt` and `NextRunAt` are both set on the target status:  
   `NextRunAt` is at or after `LastFinishedAt` (finish-based: not strictly before finish).
4. Prefer: when both parseable, `NextRunAt >= LastFinishedAt + 1s - 200ms` slack.

## Side Effects

- Marker file receives append lines from scheduled runs.
- History grows with recent runs.

## Errors

- Task never runs within poll window.
- Next run scheduled before previous finish (start-based/wrong anchor).

## Exit Code

0 from `Run`.

```go
import (
	"testing"
	"time"
)

func Assert(t *testing.T, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("Run error: %v body=%s", err, resp.Body)
	}
	if resp.ActionError != "" {
		t.Fatalf("create failed: %s", resp.ActionError)
	}
	if resp.RunCount < 2 && resp.MarkerLines < 2 {
		t.Fatalf("want ≥2 runs (history=%d markerLines=%d history=%+v)",
			resp.RunCount, resp.MarkerLines, resp.History)
	}
	if resp.Target == nil {
		t.Fatal("target status missing")
	}
	fin := resp.Target.LastFinishedAt
	next := resp.Target.NextRunAt
	if fin != "" && next != "" {
		ft, err1 := parseRFC3339(fin)
		nt, err2 := parseRFC3339(next)
		if err1 != nil || err2 != nil {
			t.Fatalf("parse times finish=%q next=%q err1=%v err2=%v", fin, next, err1, err2)
		}
		// Finish-based: next must not be before finish; ideally ≥ finish+1s (interval).
		if nt.Before(ft.Add(-200 * time.Millisecond)) {
			t.Fatalf("nextRunAt %v before lastFinishedAt %v (not finish-based)", nt, ft)
		}
		minNext := ft.Add(1 * time.Second).Add(-500 * time.Millisecond)
		if nt.Before(minNext) {
			t.Fatalf("nextRunAt %v too soon after finish %v (want ≈ finish+1s interval)", nt, ft)
		}
	}
}
```
