## Expected

1. Second GET is `from_cache` with the same live session.
2. Exactly one deep Capture (GET does not start an incremental probe).
3. No layout probe (`LayoutCalls == 0`).

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
	if !resp.FromCache {
		t.Fatal("second GET must be from_cache")
	}
	if resp.CachedAt == "" {
		t.Fatal("cached_at empty")
	}
	if resp.SessionCount != 1 {
		t.Fatalf("sessions=%d", resp.SessionCount)
	}
	if resp.CaptureCalls != 1 {
		t.Fatalf("CaptureCalls=%d want 1 (GET without refresh must not recapture)", resp.CaptureCalls)
	}
	if resp.LayoutCalls != 0 {
		t.Fatalf("LayoutCalls=%d want 0 (GET without refresh is not incremental)", resp.LayoutCalls)
	}
}
```

