## Expected

1. First frame is last-good: `from_cache`, has `sess-a`, no `sess-b`.
2. Final frame has both sessions.
3. `sess-a` cwd is the original last-good path (not `/tmp/recaptured-a` from a full recapture).
4. `sess-b` cwd comes from deep capture (`/tmp/other`), not the IDs-only layout snap.
5. A layout probe ran.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.FirstFrameFromCache || resp.FirstFrameSessionCount < 1 {
		t.Fatalf("first frame from_cache=%v sessions=%d want last-good", resp.FirstFrameFromCache, resp.FirstFrameSessionCount)
	}
	if resp.FirstFrameHasSessionB {
		t.Fatal("first frame is last-good; sess-b must not appear yet")
	}
	if !resp.HasSessionA || !resp.HasSessionB || resp.SessionCount < 2 {
		t.Fatalf("final must add sess-b sessions=%d hasA=%v hasB=%v", resp.SessionCount, resp.HasSessionA, resp.HasSessionB)
	}
	if resp.SessionACwd != "/Users/xhd2015/proj/ai-critic" {
		t.Fatalf("sess-a cwd=%q want last-good (not /tmp/recaptured-a)", resp.SessionACwd)
	}
	if resp.SessionBCwd != "/tmp/other" {
		t.Fatalf("sess-b cwd=%q want deep-captured /tmp/other", resp.SessionBCwd)
	}
	if resp.LayoutCalls < 1 {
		t.Fatalf("LayoutCalls=%d want >= 1", resp.LayoutCalls)
	}
}
```
