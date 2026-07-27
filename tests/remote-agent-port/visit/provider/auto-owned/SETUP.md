# Scenario

**Feature**: auto provider selects owned when available

```
Start(auto) + owned Available -> provider cloudflare_owned
```

## Steps

1. OwnedAvailable=true, QuickAvailable=true, Provider=auto.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-start"
	req.Port = defaultTestPort
	req.Provider = "auto"
	enableOwnedQuick(req, true, true)
	return nil
}
```
