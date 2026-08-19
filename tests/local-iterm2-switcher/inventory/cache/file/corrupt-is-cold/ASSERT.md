## Expected

1. No harness/handler panic (err nil; stream returns).
2. Cold path: first frame not `from_cache`, 0 live sessions; then capture with sess-a.
3. One deep Capture.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatalf("corrupt CachePath must not fail Run: %v", err)
	}
	if resp.StatusCode != 0 && resp.StatusCode != 200 {
		// SSE stream should complete 200; empty StatusCode only if fillHTTP skipped
		t.Fatalf("status=%d body=%s (corrupt file must not 500)", resp.StatusCode, resp.Body)
	}
	if resp.StreamFrames < 2 {
		t.Fatalf("StreamFrames=%d want seed then capture (corrupt treated as cold)", resp.StreamFrames)
	}
	if resp.FirstFrameFromCache {
		t.Fatal("corrupt CachePath: first frame must not be from_cache")
	}
	if resp.FirstFrameSessionCount != 0 {
		t.Fatalf("cold first frame sessions=%d want 0", resp.FirstFrameSessionCount)
	}
	if resp.SessionCount < 1 {
		t.Fatal("stream should still capture fixture sess-a after corrupt-file cold miss")
	}
	if resp.CaptureCalls != 1 {
		t.Fatalf("CaptureCalls=%d want 1", resp.CaptureCalls)
	}
}
```
