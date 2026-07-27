# Scenario

**Feature**: explicit owned fails when unavailable

```
Start(owned) + owned Available=false -> error
```

## Steps

1. Provider=owned; OwnedAvailable=false; QuickAvailable=true.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-start"
	req.Port = defaultTestPort
	req.Provider = "owned"
	enableOwnedQuick(req, false, true)
	return nil
}
```
