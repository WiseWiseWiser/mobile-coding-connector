# Scenario

**Feature**: GET /inventory/stream

```
# cold: seed desktop headings, then full deep capture
GET /inventory/stream -> seed (0 sessions) -> Capture -> filled inventory

# warm: last-good first, then incremental layout-diff
GET /inventory -> memory
GET /inventory/stream -> last-good from_cache -> Layout -> merge
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
	return nil
}
```
