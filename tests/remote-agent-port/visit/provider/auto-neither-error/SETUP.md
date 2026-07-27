# Scenario

**Feature**: auto errors when no provider available

```
Start(auto) + neither Available -> error
```

## Steps

1. Both Available=false; Provider=auto.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-start"
	req.Port = defaultTestPort
	req.Provider = "auto"
	enableOwnedQuick(req, false, false)
	return nil
}
```
