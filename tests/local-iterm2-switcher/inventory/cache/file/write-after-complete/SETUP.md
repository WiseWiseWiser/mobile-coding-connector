# Scenario

**Feature**: cold GET success writes complete last-good to CachePath

```
# no seed file, empty RAM
new Handler
GET /inventory -> Capture (sess-a complete)
  -> CachePath exists and contains sess-a
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "inventory"
	req.ITermRunning = true
	req.WindowSpace = 1
	// no SeedCacheJSON: file starts missing
	return nil
}
```
