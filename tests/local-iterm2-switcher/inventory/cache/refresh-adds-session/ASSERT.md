## Expected

1. `?refresh=1` is a full deep recapture (`from_cache` false, Capture twice).
2. Second snap includes `sess-b`.
3. Refresh is not the incremental layout path (`LayoutCalls == 0`).

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Assert(t *testing.T, _ *session.Doctest, req *Request, resp *Response, err error) {
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d body=%s", resp.StatusCode, resp.Body)
	}
	if resp.FromCache {
		t.Fatal("refresh=1 must recapture, not from_cache")
	}
	if !resp.HasSessionB {
		t.Fatal("refresh must pick up sess-b")
	}
	if resp.CaptureCalls < 2 {
		t.Fatalf("CaptureCalls=%d want >= 2 (full recapture)", resp.CaptureCalls)
	}
	if resp.LayoutCalls != 0 {
		t.Fatalf("LayoutCalls=%d want 0 (?refresh=1 is full recapture, not layout-diff)", resp.LayoutCalls)
	}
}
```

