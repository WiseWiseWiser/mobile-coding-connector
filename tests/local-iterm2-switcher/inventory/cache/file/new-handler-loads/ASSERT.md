## Expected

1. Stream first inventory frame is last-good from the file: `from_cache`, sess-a, session count ≥ 1.
2. GET without refresh is `from_cache` with live sess-a.
3. No deep Capture (`CaptureCalls == 0`) — file load warms RAM.

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
		t.Fatal("stream must emit at least the last-good inventory frame from file")
	}
	if !resp.FirstFrameFromCache {
		t.Fatal("new Handler with good CachePath: first stream frame must be from_cache")
	}
	if resp.FirstFrameSessionCount < 1 {
		t.Fatalf("first frame sessions=%d want ≥ 1 (file last-good, not empty seed)", resp.FirstFrameSessionCount)
	}
	if !resp.FirstFrameHasSessionA {
		t.Fatal("first frame must have sess-a from file last-good")
	}
	if !resp.FromCache {
		t.Fatal("GET without refresh must be from_cache after file load")
	}
	if !resp.HasSessionA {
		t.Fatal("GET must still have sess-a")
	}
	if resp.CaptureCalls != 0 {
		t.Fatalf("CaptureCalls=%d want 0 (warm from file, no Capture)", resp.CaptureCalls)
	}
}
```
