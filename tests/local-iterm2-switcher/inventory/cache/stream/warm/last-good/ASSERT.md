## Expected

1. First inventory SSE frame is last-good: `from_cache`, at least one live session.
2. First frame still has `sess-a` (empty seed wipe is forbidden).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StreamFrames < 1 {
		t.Fatal("warm stream must emit at least the last-good inventory frame")
	}
	if !resp.FirstFrameFromCache {
		t.Fatal("warm stream first inventory frame must be last-good from_cache")
	}
	if resp.FirstFrameSessionCount < 1 {
		t.Fatalf("first frame sessions=%d (empty seed wipe is forbidden)", resp.FirstFrameSessionCount)
	}
	if !resp.FirstFrameHasSessionA {
		t.Fatal("first frame must still have sess-a")
	}
}
```
