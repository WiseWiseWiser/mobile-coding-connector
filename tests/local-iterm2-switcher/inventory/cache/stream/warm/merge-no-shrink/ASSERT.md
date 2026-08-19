## Expected

1. First frame is last-good (`from_cache`, both sessions).
2. Minimum live-session count across **all** inventory frames is at least last-good (2).
3. Final frame still has both `sess-a` and `sess-b`.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if !resp.FirstFrameFromCache || resp.FirstFrameSessionCount < 2 {
		t.Fatalf("first frame last-good from_cache=%v sessions=%d", resp.FirstFrameFromCache, resp.FirstFrameSessionCount)
	}
	if resp.MinFrameSessionCount < 2 {
		t.Fatalf("min sessions across frames=%d (seed wipe or prefix-of-windows publish shrinks last-good)", resp.MinFrameSessionCount)
	}
	if resp.SessionCount < 2 || !resp.HasSessionA || !resp.HasSessionB {
		t.Fatalf("final still a+b sessions=%d a=%v b=%v", resp.SessionCount, resp.HasSessionA, resp.HasSessionB)
	}
}
```
