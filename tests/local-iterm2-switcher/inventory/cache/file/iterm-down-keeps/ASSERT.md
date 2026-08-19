## Expected

1. GET completes (200) with iTerm down semantics (no panic).
2. Cache file still exists and still contains sess-a (not replaced by empty last-good).

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
	if !resp.CacheFileExists {
		t.Fatal("iTerm down must keep the CachePath file")
	}
	if !resp.CacheFileHasSessionA {
		t.Fatal("iTerm down must not overwrite last-good: file must still have sess-a")
	}
	if resp.CacheFileSessionCount < 1 {
		t.Fatalf("CacheFileSessionCount=%d want ≥ 1 (keep last-good)", resp.CacheFileSessionCount)
	}
}
```
