# Scenario

**Feature**: iTerm down does not overwrite last-good file with empty inventory

```
# durable last-good already on disk
SeedCacheJSON (sess-a) -> CachePath

# iTerm not running / Capture reports down
ITermRunning=false
GET /inventory -> iterm_running false, 0 live sessions
  -> CachePath still has sess-a (not empty overwrite)
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "inventory"
	req.ITermRunning = false
	req.WindowSpace = 1
	req.SeedCacheJSON = fixtureLastGoodCacheJSON()
	return nil
}
```
