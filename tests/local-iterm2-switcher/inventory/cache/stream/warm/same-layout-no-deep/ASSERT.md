## Expected

1. Exactly one deep Capture (the warm GET). Stream must not recapture known IDs.
2. At least one layout-only probe (always incremental; not TTL skip).
3. Final `sess-a` cwd is last-good (layout IDs-only must not wipe cwd).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.CaptureCalls != 1 {
		t.Fatalf("CaptureCalls=%d want 1 (warm GET only; no deep recapture of known IDs)", resp.CaptureCalls)
	}
	if resp.LayoutCalls < 1 {
		t.Fatalf("LayoutCalls=%d want >= 1 (always incremental layout-diff, not TTL skip)", resp.LayoutCalls)
	}
	if resp.SessionACwd != "/Users/xhd2015/proj/ai-critic" {
		t.Fatalf("sess-a cwd=%q was overwritten (known IDs must keep last-good cwd)", resp.SessionACwd)
	}
	if resp.SessionCount != 1 || !resp.HasSessionA {
		t.Fatalf("final sessions=%d hasA=%v want sess-a only", resp.SessionCount, resp.HasSessionA)
	}
}
```
