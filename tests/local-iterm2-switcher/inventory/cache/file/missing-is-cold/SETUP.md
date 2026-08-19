# Scenario

**Feature**: missing CachePath file is cold (new Handler, no seed file)

```
# no iterm-inventory-cache.json
new Handler (empty RAM, no file)
GET /inventory/stream
  -> first frame: seed (0 live sessions), not from_cache
  -> later: full Capture with sess-a
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "stream"
	req.ITermRunning = true
	req.WindowSpace = 1
	// SeedCacheJSON empty: CachePath file absent
	return nil
}
```
