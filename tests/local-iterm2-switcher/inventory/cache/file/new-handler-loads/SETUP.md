# Scenario

**Feature**: new Handler loads complete last-good from CachePath (empty RAM)

```
# prior process left a complete last-good file
SeedCacheJSON (sess-a) -> CachePath

# new Handler: no prior in-process GET
Handler (empty RAM) -> load file
GET /inventory/stream -> first frame from_cache + sess-a
GET /inventory (no refresh) -> from_cache, CaptureCalls=0
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
	req.SeedCacheJSON = fixtureLastGoodCacheJSON()
	req.DoGETAfterStream = true
	return nil
}
```
