# Scenario

**Feature**: second GET hits the in-memory inventory cache

```
GET /inventory -> Capture
GET /inventory -> from_cache, no second Capture
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
	req.DoSecondGET = true
	return nil
}
```
