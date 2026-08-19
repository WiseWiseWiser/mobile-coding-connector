# Scenario

**Feature**: warm stream first inventory frame is last-good

```
GET /inventory -> sess-a cached
GET /inventory/stream
  -> first inventory frame: from_cache, sess-a still present
  -> must not emit empty desktop seed that wipes live sessions
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
