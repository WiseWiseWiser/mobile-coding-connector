# Scenario

**Feature**: cold stream seeds desktop headings before full capture

```
GET /inventory/stream (no memory)
  -> first inventory frame: desktops, 0 live sessions
  -> later frame: fixture sess-a from full Capture
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
