# Scenario

**Feature**: unreadable/corrupt CachePath is cold (must not panic)

```
# CachePath contains invalid JSON
SeedCacheJSON = "{not json"
new Handler
GET /inventory/stream
  -> same as missing: seed then Capture (not from_cache first)
  -> must not panic / 500
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
	req.SeedCacheJSON = `{not json`
	return nil
}
```
