# Scenario

**Feature**: warm layout-diff adds a new session without recapturing known IDs

```
GET /inventory -> sess-a last-good
GET /inventory/stream
  -> first frame: from_cache sess-a only
  -> Layout sees sess-a + sess-b
  -> deep-capture sess-b only; keep sess-a cwd
  -> final frame has both
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "stream"
	req.DoSecondGET = true
	req.SecondSnapB = true
	return nil
}
```
