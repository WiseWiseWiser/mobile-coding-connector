# Scenario

**Feature**: explicit quick used even when owned available

```
Start(quick) + both Available -> cloudflare_quick
```

## Steps

1. Provider=quick; both available.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-start"
	req.Port = defaultTestPort
	req.Provider = "quick"
	enableOwnedQuick(req, true, true)
	return nil
}
```
