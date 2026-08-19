# Scenario

**Feature**: stream merge publish never shrinks below last-good

```
GET /inventory -> two windows (sess-a, sess-b) last-good
GET /inventory/stream
  -> every inventory frame has SessionCount >= 2
  -> prefix-of-windows publish is forbidden
```

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)
func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "stream"
	req.DoSecondGET = true
	req.UseTwoWindowSnap = true
	return nil
}
```
