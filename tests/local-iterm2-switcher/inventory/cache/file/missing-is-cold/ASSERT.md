## Expected

1. At least two inventory SSE frames (seed, then capture).
2. First frame is cold seed: not `from_cache`, 0 live sessions, desktops listed.
3. Final frame includes fixture sess-a.
4. One full deep Capture.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StreamFrames < 2 {
		t.Fatalf("StreamFrames=%d want seed then at least one capture frame", resp.StreamFrames)
	}
	if resp.FirstFrameFromCache {
		t.Fatal("missing CachePath: first frame must not be from_cache")
	}
	if resp.FirstFrameSessionCount != 0 {
		t.Fatalf("cold first frame sessions=%d want 0 (desktop seed)", resp.FirstFrameSessionCount)
	}
	if resp.FirstFrameDesktopCount < 1 && resp.DesktopCount < 1 {
		t.Fatal("seed/final inventory must list desktops")
	}
	if resp.SessionCount < 1 {
		t.Fatal("stream should include the fixture session after capture")
	}
	if resp.CaptureCalls != 1 {
		t.Fatalf("CaptureCalls=%d want 1 (one full cold capture)", resp.CaptureCalls)
	}
}
```
