# Scenario

**Feature**: stop visit by session id

```
Start -> Stop(id) -> empty list
```

## Steps

1. Op=visit-stop; StopTarget left empty (uses id from Start).

```go
import (
	"testing"
	"time"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-stop"
	req.Port = defaultTestPort
	req.Provider = "quick"
	enableOwnedQuick(req, true, true)
	req.Idle = 10 * time.Minute
	req.StopTarget = ""
	return nil
}
```
