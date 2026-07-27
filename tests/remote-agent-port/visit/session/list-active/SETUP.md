# Scenario

**Feature**: visit list shows active sessions

```
Start -> List -> one session with public URL and port
```

## Steps

1. Op=visit-list with Port set so Start runs first.

```go
import (
	"testing"
	"time"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-list"
	req.Port = defaultTestPort
	req.Provider = "owned"
	enableOwnedQuick(req, true, true)
	req.Idle = 10 * time.Minute
	return nil
}
```
