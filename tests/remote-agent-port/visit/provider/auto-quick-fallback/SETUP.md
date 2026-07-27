# Scenario

**Feature**: auto falls back to quick when owned unavailable

```
Start(auto) + owned down + quick up -> cloudflare_quick
```

## Steps

1. OwnedAvailable=false, QuickAvailable=true, Provider=auto.

```go
import (
	"testing"
	"github.com/xhd2015/doctest/session"
)

func Setup(t *testing.T, _ *session.Doctest, req *Request) error {
	req.Op = "visit-start"
	req.Port = defaultTestPort
	req.Provider = "auto"
	enableOwnedQuick(req, false, true)
	return nil
}
```
