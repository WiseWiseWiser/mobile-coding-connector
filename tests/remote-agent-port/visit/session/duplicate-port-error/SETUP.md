# Scenario

**Feature**: second visit on same port while active errors

```
Start(port) -> Start(port) -> error
```

## Steps

1. Op=visit-duplicate.

```go
import (
	"testing"
	"time"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-duplicate"
	req.Port = defaultTestPort
	req.Provider = "auto"
	enableOwnedQuick(req, true, true)
	req.Idle = 10 * time.Minute
	return nil
}
```
