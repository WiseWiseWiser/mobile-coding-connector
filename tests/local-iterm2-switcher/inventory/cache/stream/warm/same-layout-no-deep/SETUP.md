# Scenario

**Feature**: warm same-layout stream probes layout, does not deep-recapture known IDs

```
GET /inventory -> Capture sess-a
GET /inventory/stream (same window/tab/session IDs)
  -> Layout probe runs
  -> Capture is not called again for sess-a
  -> sess-a cwd stays last-good
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "stream"
	req.DoSecondGET = true
	return nil
}
```
