# Scenario

**Feature**: warm stream after daemon memory exists

```
GET /inventory -> last-good in daemon RAM
GET /inventory/stream -> first frame last-good, then incremental probe
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
	req.DoSecondGET = true
	return nil
}
```
