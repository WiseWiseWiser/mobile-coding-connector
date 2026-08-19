## Expected

1. First frame is last-good with both `sess-a` and `sess-b`.
2. No intermediate frame has fewer live sessions than last-good (2).
3. Final frame has `sess-a` only (`sess-b` dropped).
4. A layout probe ran.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.FirstFrameSessionCount < 2 || !resp.FirstFrameHasSessionA || !resp.FirstFrameHasSessionB {
		t.Fatalf("first frame must be last-good a+b count=%d a=%v b=%v", resp.FirstFrameSessionCount, resp.FirstFrameHasSessionA, resp.FirstFrameHasSessionB)
	}
	if resp.MinNonFinalSessionCount < 2 {
		t.Fatalf("intermediate min sessions=%d (must not drop below last-good until final)", resp.MinNonFinalSessionCount)
	}
	if resp.SessionCount != 1 || !resp.HasSessionA || resp.HasSessionB {
		t.Fatalf("final must be sess-a only sessions=%d a=%v b=%v", resp.SessionCount, resp.HasSessionA, resp.HasSessionB)
	}
	if resp.LayoutCalls < 1 {
		t.Fatalf("LayoutCalls=%d want >= 1", resp.LayoutCalls)
	}
}
```
