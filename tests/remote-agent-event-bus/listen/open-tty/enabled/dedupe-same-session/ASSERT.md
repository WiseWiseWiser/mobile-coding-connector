## Expected

1. Exit code 0.
2. All three events printed (`agent.tty.started` appears thrice or three event ids).
3. OpenTTYSession called exactly twice: first A, then B (no second open for A).

## Errors

- Second open for same session_id.
- Distinct session_id B never opened.
- Wrong call order.

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
	typeCount := strings.Count(out, fixtureTypeTTY)
	if typeCount < 3 {
		// allow id-based proof if type string collapsed
		hasAll := strings.Contains(out, fixtureEventID1) &&
			strings.Contains(out, fixtureEventID2) &&
			strings.Contains(out, fixtureEventID3)
		if !hasAll {
			t.Fatalf("expected three printed tty events (typeCount=%d); stdout:\n%s", typeCount, out)
		}
	}
	want := []string{fixtureSessionIDA, fixtureSessionIDB}
	if len(resp.OpenTTYSessionIDs) != len(want) {
		t.Fatalf("OpenTTYSession calls want %v got %v; combined:\n%s",
			want, resp.OpenTTYSessionIDs, resp.Combined)
	}
	for i := range want {
		if resp.OpenTTYSessionIDs[i] != want[i] {
			t.Fatalf("OpenTTYSession[%d] want %q got %q; full=%v",
				i, want[i], resp.OpenTTYSessionIDs[i], resp.OpenTTYSessionIDs)
		}
	}
}
```
