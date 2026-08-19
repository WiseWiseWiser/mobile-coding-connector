## Expected

1. GET succeeds with live sess-a.
2. Cache file exists after complete probe.
3. Cache file JSON contains sess-a (session count ≥ 1).

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
	if !resp.HasSessionA || resp.SessionCount < 1 {
		t.Fatal("GET must return fixture sess-a")
	}
	if !resp.CacheFileExists {
		t.Fatal("after complete GET, CachePath file must exist")
	}
	if !resp.CacheFileHasSessionA {
		t.Fatal("cache file must contain sess-a")
	}
	if resp.CacheFileSessionCount < 1 {
		t.Fatalf("CacheFileSessionCount=%d want ≥ 1", resp.CacheFileSessionCount)
	}
}
```
