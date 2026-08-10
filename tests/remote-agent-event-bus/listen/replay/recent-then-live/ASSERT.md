## Expected

1. Exit code 0.
2. Stdout includes all three event ids (or payloads n=1,2,3) — two replayed + one live.
3. Order is chronological: replay oldest-first, then live (id1 before id2 before id3).

## Errors

- Only live event without replay.
- Wrong order.
- Missing ids/payloads.

## Exit Code

0.

```go
import (
	"strings"
	"testing"

	"github.com/xhd2015/doctest/session"
)

func Assert(t *testing.T, d *session.Doctest, req *Request, resp *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit %d; combined:\n%s", resp.ExitCode, resp.Combined)
	}
	out := resp.Stdout
	// Prefer stable ids; fall back to payload markers.
	markers := []string{fixtureEventID1, fixtureEventID2, fixtureEventID3}
	payloads := []string{`"n":1`, `"n":2`, `"n":3`}
	for i, id := range markers {
		if strings.Contains(out, id) {
			continue
		}
		if strings.Contains(out, payloads[i]) {
			continue
		}
		t.Fatalf("missing replay/live marker %q or %q; stdout:\n%s", id, payloads[i], out)
	}
	// Order: id1 before id2 before id3 when present as ids.
	if strings.Contains(out, fixtureEventID1) && strings.Contains(out, fixtureEventID2) && strings.Contains(out, fixtureEventID3) {
		i1 := strings.Index(out, fixtureEventID1)
		i2 := strings.Index(out, fixtureEventID2)
		i3 := strings.Index(out, fixtureEventID3)
		if !(i1 < i2 && i2 < i3) {
			t.Fatalf("expected chronological order id1 < id2 < id3; stdout:\n%s", out)
		}
	}
}
```
