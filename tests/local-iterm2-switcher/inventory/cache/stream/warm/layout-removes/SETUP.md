# Scenario

**Feature**: warm layout-diff drops a gone session only on the final frame

```
GET /inventory -> sess-a + sess-b last-good
GET /inventory/stream
  -> Layout sees sess-a only
  -> intermediate frames keep both (no early drop, no wipe to 0)
  -> final frame has sess-a only
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "stream"
	req.DoSecondGET = true
	req.FirstSnapAB = true
	req.SecondSnapAOnly = true
	return nil
}
```
